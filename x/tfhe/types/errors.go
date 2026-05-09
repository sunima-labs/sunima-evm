package types

import "errors"

var (
	ErrCiphertextNotFound      = errors.New("ciphertext not found")
	ErrInvalidCiphertext       = errors.New("invalid ciphertext")
	ErrAttesterNotRegistered   = errors.New("attester not registered")
	ErrInsufficientAttestations = errors.New("insufficient attestations for quorum")
	ErrInvalidAttestationSig   = errors.New("invalid attestation signature")
	ErrUnauthorized            = errors.New("sender not authorized for this ciphertext")
)
