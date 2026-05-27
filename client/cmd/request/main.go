package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/prdai/aws-nitro-enclaves-remote-attestation-verification-rnd/client/internal/attestation"
)

type attestationResponse struct {
	Status              string `json:"status"`
	Message             string `json:"message"`
	AttestationDocument string `json:"attestation_document"`
	Encoding            string `json:"encoding"`
	Time                string `json:"time"`
}

func main() {
	target := flag.String("url", os.Getenv("EC2_ATTESTATION_URL"), "EC2 attestation relay URL; defaults to EC2_ATTESTATION_URL")
	timeout := flag.Duration("timeout", 10*time.Second, "request timeout")
	rootCertPath := flag.String("root-cert", os.Getenv("AWS_NITRO_ROOT_CERT"), "AWS Nitro Enclaves root certificate PEM path; defaults to AWS_NITRO_ROOT_CERT")
	nonceValue := flag.String("nonce", "", "base64url nonce to request and verify; generated when empty")
	flag.Parse()

	if *target == "" {
		fmt.Fprintln(os.Stderr, "missing EC2 attestation relay URL: pass -url or set EC2_ATTESTATION_URL")
		os.Exit(1)
	}
	if *rootCertPath == "" {
		fmt.Fprintln(os.Stderr, "missing AWS Nitro root certificate: pass -root-cert or set AWS_NITRO_ROOT_CERT")
		os.Exit(1)
	}

	rootCert, err := os.ReadFile(*rootCertPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read root certificate: %v\n", err)
		os.Exit(1)
	}

	nonce, err := nonceFromFlag(*nonceValue)
	if err != nil {
		fmt.Fprintf(os.Stderr, "prepare nonce: %v\n", err)
		os.Exit(1)
	}

	requestURL, err := url.Parse(*target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse URL: %v\n", err)
		os.Exit(1)
	}
	query := requestURL.Query()
	query.Set("nonce", base64.RawURLEncoding.EncodeToString(nonce))
	requestURL.RawQuery = query.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build request: %v\n", err)
		os.Exit(1)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "send request: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close response body: %v\n", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("status: %s\n", resp.Status)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		fmt.Print(string(body))
		os.Exit(1)
	}

	var envelope attestationResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		fmt.Print(string(body))
		return
	}

	fmt.Printf("message: %s\n", envelope.Message)
	fmt.Printf("nonce: %s\n", base64.RawURLEncoding.EncodeToString(nonce))
	if envelope.AttestationDocument == "" {
		fmt.Print(string(body))
		return
	}

	rawDoc, err := base64.StdEncoding.DecodeString(envelope.AttestationDocument)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode attestation document: %v\n", err)
		os.Exit(1)
	}

	result, err := attestation.Verify(rawDoc, attestation.Expected{
		RootCertificatePEM: rootCert,
		Nonce:              nonce,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "verify attestation document: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("verification: ok")
	fmt.Printf("module_id: %s\n", result.ModuleID)
	fmt.Printf("digest: %s\n", result.Digest)
	fmt.Printf("timestamp: %s\n", result.Timestamp.Format(time.RFC3339))
	fmt.Printf("root_sha256: %s\n", result.RootFingerprint)
	for index, value := range result.PCRs {
		fmt.Printf("pcr%d: %s\n", index, hex.EncodeToString(value))
	}
}

func nonceFromFlag(value string) ([]byte, error) {
	if value != "" {
		nonce, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			return nil, fmt.Errorf("decode base64url nonce: %w", err)
		}
		return nonce, nil
	}

	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate random nonce: %w", err)
	}
	return nonce, nil
}
