package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

const defaultAddr = ":8080"
const defaultEnclaveCID = 16
const defaultEnclavePort = 5000

type response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Time    string `json:"time"`
}

type attestationResponse struct {
	Status              string `json:"status"`
	Message             string `json:"message"`
	AttestationDocument string `json:"attestation_document,omitempty"`
	Encoding            string `json:"encoding,omitempty"`
	Time                string `json:"time"`
}

type attestationProvider struct {
	enclaveCID  uint32
	enclavePort uint32
	dialContext func(context.Context, uint32, uint32) (net.Conn, error)
}

type enclaveRequest struct {
	Nonce string `json:"nonce"`
}

type enclaveResponse struct {
	Status              string `json:"status"`
	Message             string `json:"message,omitempty"`
	AttestationDocument string `json:"attestation_document,omitempty"`
	Encoding            string `json:"encoding,omitempty"`
}

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	provider := attestationProvider{
		enclaveCID:  envUint32("ENCLAVE_CID", defaultEnclaveCID),
		enclavePort: envUint32("ENCLAVE_PORT", defaultEnclavePort),
		dialContext: dialEnclave,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealth)
	mux.HandleFunc("GET /attestation", provider.handleAttestation)
	mux.HandleFunc("GET /", handleRequest)
	mux.HandleFunc("POST /", handleRequest)

	server := &http.Server{
		Addr:              addr,
		Handler:           requestLogger(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.Info("starting public HTTP server", "addr", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, response{
		Status:  "ok",
		Message: "ec2 http server is healthy",
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, response{
		Status:  "ok",
		Message: "request received by ec2 http server",
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
}

func (p attestationProvider) handleAttestation(w http.ResponseWriter, r *http.Request) {
	nonce := r.URL.Query().Get("nonce")
	rawDoc, err := p.attestationDocument(r.Context(), nonce)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, errAttestationProviderMissing) {
			status = http.StatusNotImplemented
		}
		writeJSON(w, status, attestationResponse{
			Status:  "error",
			Message: err.Error(),
			Time:    time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	writeJSON(w, http.StatusOK, attestationResponse{
		Status:              "ok",
		Message:             "attestation document returned by enclave relay",
		AttestationDocument: base64.StdEncoding.EncodeToString(rawDoc),
		Encoding:            "base64",
		Time:                time.Now().UTC().Format(time.RFC3339),
	})
}

var errAttestationProviderMissing = errors.New("attestation provider is not configured")

func (p attestationProvider) attestationDocument(ctx context.Context, nonce string) ([]byte, error) {
	if p.dialContext == nil {
		return nil, errAttestationProviderMissing
	}
	return p.fetchFromEnclave(ctx, nonce)
}

func (p attestationProvider) fetchFromEnclave(ctx context.Context, nonce string) ([]byte, error) {
	conn, err := p.dialContext(ctx, p.enclaveCID, p.enclavePort)
	if err != nil {
		return nil, fmt.Errorf("connect to enclave over vsock: %w", err)
	}
	defer closeConn(conn)

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return nil, fmt.Errorf("set enclave connection deadline: %w", err)
		}
	}

	if err := json.NewEncoder(conn).Encode(enclaveRequest{Nonce: nonce}); err != nil {
		return nil, fmt.Errorf("write enclave request: %w", err)
	}

	var envelope enclaveResponse
	if err := json.NewDecoder(conn).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode enclave response: %w", err)
	}
	if envelope.Status != "ok" {
		return nil, fmt.Errorf("enclave returned error: %s", envelope.Message)
	}
	if envelope.AttestationDocument == "" {
		return nil, errors.New("enclave response did not include attestation_document")
	}
	rawDoc, err := base64.StdEncoding.DecodeString(envelope.AttestationDocument)
	if err != nil {
		return nil, fmt.Errorf("decode enclave attestation_document: %w", err)
	}
	return rawDoc, nil
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

func closeConn(conn net.Conn) {
	if err := conn.Close(); err != nil {
		slog.Error("failed to close enclave connection", "error", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("failed to write response", "error", err)
	}
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("handled request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"duration", time.Since(start).String(),
		)
	})
}
