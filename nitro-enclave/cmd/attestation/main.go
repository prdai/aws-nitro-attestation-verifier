package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"time"
)

const defaultVsockPort = 5000

type request struct {
	Nonce string `json:"nonce"`
}

type response struct {
	Status              string `json:"status"`
	Message             string `json:"message"`
	AttestationDocument string `json:"attestation_document,omitempty"`
	Encoding            string `json:"encoding,omitempty"`
	Nonce               string `json:"nonce,omitempty"`
	Time                string `json:"time"`
}

type attester interface {
	AttestationDocument(nonce []byte) ([]byte, error)
}

func main() {
	port := envUint32("VSOCK_PORT", defaultVsockPort)
	listener, err := listenVsock(port)
	if err != nil {
		slog.Error("failed to listen on vsock", "port", port, "error", err)
		os.Exit(1)
	}
	defer closeListener(listener)

	provider := nsmAttester{}
	slog.Info("starting enclave attestation vsock server", "port", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Error("accept vsock connection", "error", err)
			continue
		}
		go func() {
			defer closeConn(conn)
			if err := handleConnection(conn, provider); err != nil {
				slog.Error("handle attestation request", "error", err)
			}
		}()
	}
}

func handleConnection(conn net.Conn, provider attester) error {
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("set connection deadline: %w", err)
	}

	var req request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		return fmt.Errorf("decode request: %w", err)
	}

	nonce, err := base64.RawURLEncoding.DecodeString(req.Nonce)
	if err != nil {
		return writeResponse(conn, response{
			Status:  "error",
			Message: fmt.Sprintf("decode base64url nonce: %v", err),
			Time:    time.Now().UTC().Format(time.RFC3339),
		})
	}

	rawDoc, err := provider.AttestationDocument(nonce)
	if err != nil {
		return writeResponse(conn, response{
			Status:  "error",
			Message: err.Error(),
			Time:    time.Now().UTC().Format(time.RFC3339),
		})
	}

	return writeResponse(conn, response{
		Status:              "ok",
		Message:             "attestation document returned by nitro enclave",
		AttestationDocument: base64.StdEncoding.EncodeToString(rawDoc),
		Encoding:            "base64",
		Nonce:               req.Nonce,
		Time:                time.Now().UTC().Format(time.RFC3339),
	})
}

func writeResponse(conn net.Conn, value response) error {
	if err := json.NewEncoder(conn).Encode(value); err != nil {
		return fmt.Errorf("write response: %w", err)
	}
	return nil
}

func envUint32(name string, fallback uint32) uint32 {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		slog.Warn("invalid uint32 env value, using fallback", "name", name, "value", value, "fallback", fallback)
		return fallback
	}
	return uint32(parsed)
}

func closeListener(listener net.Listener) {
	if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		slog.Error("failed to close vsock listener", "error", err)
	}
}

func closeConn(conn net.Conn) {
	if err := conn.Close(); err != nil {
		slog.Error("failed to close vsock connection", "error", err)
	}
}
