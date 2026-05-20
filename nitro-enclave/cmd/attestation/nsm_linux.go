//go:build linux

package main

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/cyclicalvc/nitro-enclaves-sdk/nsm"
	nsmrequest "github.com/cyclicalvc/nitro-enclaves-sdk/nsm/request"
)

type nsmAttester struct{}

func (nsmAttester) AttestationDocument(nonce []byte) ([]byte, error) {
	if len(nonce) > 512 {
		return nil, fmt.Errorf("nonce is %d bytes, max 512", len(nonce))
	}

	session, err := nsm.OpenDefaultSession()
	if err != nil {
		return nil, fmt.Errorf("open NSM session: %w", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			slog.Error("close NSM session", "error", err)
		}
	}()

	response, err := session.Send(&nsmrequest.Attestation{Nonce: nonce})
	if err != nil {
		return nil, fmt.Errorf("request NSM attestation document: %w", err)
	}
	if response.Error != "" {
		return nil, fmt.Errorf("NSM error: %s", response.Error)
	}
	if response.Attestation == nil || len(response.Attestation.Document) == 0 {
		return nil, errors.New("NSM response did not include attestation document")
	}
	return response.Attestation.Document, nil
}
