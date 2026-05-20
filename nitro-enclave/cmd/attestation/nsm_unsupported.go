//go:build !linux

package main

import "errors"

type nsmAttester struct{}

func (nsmAttester) AttestationDocument(_ []byte) ([]byte, error) {
	return nil, errors.New("NSM attestation is only available inside linux nitro enclaves")
}
