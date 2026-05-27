package attestation

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/fxamacker/cbor/v2"
)

const (
	coseSign1Tag           = 18
	coseAlgHeaderKey       = 1
	coseAlgES256     int64 = -7
	coseAlgES384     int64 = -35
	coseAlgES512     int64 = -36
)

type Expected struct {
	RootCertificatePEM []byte
	Nonce              []byte
	PublicKey          []byte
	UserData           []byte
	PCRs               map[uint64][]byte
	Now                time.Time
}

type Result struct {
	ModuleID          string
	Digest            string
	Timestamp         time.Time
	PCRs              map[uint64][]byte
	Certificate       *x509.Certificate
	CertificateChain  []*x509.Certificate
	RootFingerprint   string
	NonceVerified     bool
	PublicKeyVerified bool
	UserDataVerified  bool
	PCRsVerified      bool
}

type coseSign1 struct {
	protected []byte
	payload   []byte
	signature []byte
}

type attestationDocument struct {
	ModuleID    string            `cbor:"module_id"`
	Timestamp   uint64            `cbor:"timestamp"`
	Digest      string            `cbor:"digest"`
	PCRs        map[uint64][]byte `cbor:"pcrs"`
	Certificate []byte            `cbor:"certificate"`
	CABundle    [][]byte          `cbor:"cabundle"`
	PublicKey   []byte            `cbor:"public_key"`
	UserData    []byte            `cbor:"user_data"`
	Nonce       []byte            `cbor:"nonce"`
}

func Verify(raw []byte, expected Expected) (*Result, error) {
	if len(expected.RootCertificatePEM) == 0 {
		return nil, errors.New("root certificate is required")
	}

	msg, err := decodeCOSESign1(raw)
	if err != nil {
		return nil, err
	}

	var doc attestationDocument
	if err := cbor.Unmarshal(msg.payload, &doc); err != nil {
		return nil, fmt.Errorf("decode attestation document: %w", err)
	}
	if len(doc.Certificate) == 0 {
		return nil, errors.New("attestation document does not contain signing certificate")
	}

	cert, chain, rootFingerprint, err := verifyCertificateChain(doc, expected)
	if err != nil {
		return nil, err
	}
	if err := verifyCOSESignature(msg, cert); err != nil {
		return nil, err
	}
	if err := verifyExpectedFields(doc, expected); err != nil {
		return nil, err
	}

	return &Result{
		ModuleID:          doc.ModuleID,
		Digest:            doc.Digest,
		Timestamp:         time.UnixMilli(int64(doc.Timestamp)).UTC(),
		PCRs:              doc.PCRs,
		Certificate:       cert,
		CertificateChain:  chain,
		RootFingerprint:   rootFingerprint,
		NonceVerified:     expected.Nonce == nil || bytes.Equal(expected.Nonce, doc.Nonce),
		PublicKeyVerified: expected.PublicKey == nil || bytes.Equal(expected.PublicKey, doc.PublicKey),
		UserDataVerified:  expected.UserData == nil || bytes.Equal(expected.UserData, doc.UserData),
		PCRsVerified:      true,
	}, nil
}

func decodeCOSESign1(raw []byte) (coseSign1, error) {
	var tagged cbor.Tag
	if err := cbor.Unmarshal(raw, &tagged); err == nil && tagged.Number == coseSign1Tag {
		encoded, err := cbor.Marshal(tagged.Content)
		if err != nil {
			return coseSign1{}, fmt.Errorf("re-encode COSE_Sign1 content: %w", err)
		}
		return decodeCOSESign1Array(encoded)
	}

	return decodeCOSESign1Array(raw)
}

func decodeCOSESign1Array(raw []byte) (coseSign1, error) {
	var parts []cbor.RawMessage
	if err := cbor.Unmarshal(raw, &parts); err != nil {
		return coseSign1{}, fmt.Errorf("decode COSE_Sign1 array: %w", err)
	}
	if len(parts) != 4 {
		return coseSign1{}, fmt.Errorf("COSE_Sign1 must have 4 elements, got %d", len(parts))
	}

	var msg coseSign1
	if err := cbor.Unmarshal(parts[0], &msg.protected); err != nil {
		return coseSign1{}, fmt.Errorf("decode protected headers: %w", err)
	}
	if err := cbor.Unmarshal(parts[2], &msg.payload); err != nil {
		return coseSign1{}, fmt.Errorf("decode payload: %w", err)
	}
	if err := cbor.Unmarshal(parts[3], &msg.signature); err != nil {
		return coseSign1{}, fmt.Errorf("decode signature: %w", err)
	}
	if len(msg.protected) == 0 {
		return coseSign1{}, errors.New("COSE_Sign1 protected headers are empty")
	}
	if len(msg.payload) == 0 {
		return coseSign1{}, errors.New("COSE_Sign1 payload is empty")
	}
	if len(msg.signature) == 0 {
		return coseSign1{}, errors.New("COSE_Sign1 signature is empty")
	}
	return msg, nil
}

func verifyCertificateChain(doc attestationDocument, expected Expected) (*x509.Certificate, []*x509.Certificate, string, error) {
	root, err := parseCertificate(expected.RootCertificatePEM)
	if err != nil {
		return nil, nil, "", fmt.Errorf("parse root certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(doc.Certificate)
	if err != nil {
		return nil, nil, "", fmt.Errorf("parse attestation certificate: %w", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(root)
	intermediates := x509.NewCertPool()
	chain := []*x509.Certificate{cert}

	for _, der := range doc.CABundle {
		ca, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, nil, "", fmt.Errorf("parse CA bundle certificate: %w", err)
		}
		chain = append(chain, ca)
		if !ca.Equal(root) {
			intermediates.AddCert(ca)
		}
	}

	now := expected.Now
	if now.IsZero() {
		now = time.Now()
	}
	if _, err := cert.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		CurrentTime:   now,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}); err != nil {
		return nil, nil, "", fmt.Errorf("verify attestation certificate chain: %w", err)
	}

	fingerprint := sha256.Sum256(root.Raw)
	return cert, chain, fmt.Sprintf("%X", fingerprint[:]), nil
}

func parseCertificate(data []byte) (*x509.Certificate, error) {
	if block, _ := pem.Decode(data); block != nil {
		data = block.Bytes
	}
	cert, err := x509.ParseCertificate(data)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func verifyCOSESignature(msg coseSign1, cert *x509.Certificate) error {
	alg, hash, err := signatureAlgorithm(msg.protected)
	if err != nil {
		return err
	}

	pub, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("attestation certificate does not contain an ECDSA public key")
	}

	sigStructure, err := cbor.Marshal([]any{"Signature1", msg.protected, []byte{}, msg.payload})
	if err != nil {
		return fmt.Errorf("build COSE Sig_structure: %w", err)
	}

	digest, err := hashBytes(hash, sigStructure)
	if err != nil {
		return err
	}
	if len(msg.signature)%2 != 0 {
		return fmt.Errorf("invalid ECDSA signature length %d", len(msg.signature))
	}

	half := len(msg.signature) / 2
	r := new(big.Int).SetBytes(msg.signature[:half])
	s := new(big.Int).SetBytes(msg.signature[half:])
	if !ecdsa.Verify(pub, digest, r, s) {
		return fmt.Errorf("verify COSE signature with algorithm %d", alg)
	}
	return nil
}

func signatureAlgorithm(protected []byte) (int64, crypto.Hash, error) {
	var headers map[int64]int64
	if err := cbor.Unmarshal(protected, &headers); err != nil {
		return 0, 0, fmt.Errorf("decode protected header map: %w", err)
	}

	alg, ok := headers[coseAlgHeaderKey]
	if !ok {
		return 0, 0, errors.New("COSE protected headers do not include algorithm")
	}

	switch alg {
	case coseAlgES256:
		return alg, crypto.SHA256, nil
	case coseAlgES384:
		return alg, crypto.SHA384, nil
	case coseAlgES512:
		return alg, crypto.SHA512, nil
	default:
		return alg, 0, fmt.Errorf("unsupported COSE algorithm %d", alg)
	}
}

func hashBytes(hash crypto.Hash, data []byte) ([]byte, error) {
	switch hash {
	case crypto.SHA256:
		sum := sha256.Sum256(data)
		return sum[:], nil
	case crypto.SHA384:
		sum := sha512.Sum384(data)
		return sum[:], nil
	case crypto.SHA512:
		sum := sha512.Sum512(data)
		return sum[:], nil
	default:
		return nil, fmt.Errorf("unsupported hash %v", hash)
	}
}

func verifyExpectedFields(doc attestationDocument, expected Expected) error {
	if expected.Nonce != nil && !bytes.Equal(expected.Nonce, doc.Nonce) {
		return errors.New("attestation nonce does not match expected nonce")
	}
	if expected.PublicKey != nil && !bytes.Equal(expected.PublicKey, doc.PublicKey) {
		return errors.New("attestation public_key does not match expected value")
	}
	if expected.UserData != nil && !bytes.Equal(expected.UserData, doc.UserData) {
		return errors.New("attestation user_data does not match expected value")
	}
	for index, want := range expected.PCRs {
		got, ok := doc.PCRs[index]
		if !ok {
			return fmt.Errorf("attestation PCR%d is missing", index)
		}
		if !bytes.Equal(want, got) {
			return fmt.Errorf("attestation PCR%d does not match expected value", index)
		}
	}
	return nil
}
