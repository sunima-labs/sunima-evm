package types

import "errors"

var (
	ErrCiphertextNotFound       = errors.New("ciphertext not found")
	ErrCiphertextAlreadyExists  = errors.New("ciphertext already exists")
	ErrInvalidCiphertext        = errors.New("invalid ciphertext")
	ErrEmptyCiphertext          = errors.New("ciphertext is empty")
	ErrEmptyOwner               = errors.New("owner is empty")
	ErrAttesterNotRegistered    = errors.New("attester not registered")
	ErrInsufficientAttestations = errors.New("insufficient attestations for quorum")
	ErrInvalidAttestationSig    = errors.New("invalid attestation signature")
	ErrUnauthorized             = errors.New("sender not authorized for this ciphertext")
	ErrInvalidOpType            = errors.New("unknown homomorphic operation type")
	ErrInvalidInputCount        = errors.New("invalid input count for operation")
	ErrServerKeyNotSet          = errors.New("server key not configured")
)
