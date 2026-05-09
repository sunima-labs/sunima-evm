// Package tfhebridge wraps the Rust tfhe-rs library via cgo.
//
// Memory contract:
//   - Buffers returned by the Rust side are allocated in Rust and MUST be
//     released by calling tfhe_free on the C side. This package always
//     copies into Go-owned []byte before freeing, so callers receive
//     normal Go slices.
//   - Every exported function is panic-safe: a Rust-side panic is
//     caught and surfaced as ErrOperation.
//
// Stage 5.1 Week 1 scope: keygen, encrypt u64, add, decrypt. No
// bootstrap-bound ops (mul, compare) yet.
package tfhebridge

/*
#cgo CFLAGS: -I${SRCDIR}/rust
#cgo LDFLAGS: -L${SRCDIR}/rust/target/release -ltfhebridge -ldl -lpthread -lm
#include <stdint.h>
#include <stddef.h>

int tfhe_keygen(
    unsigned char **out_ck_ptr, size_t *out_ck_len,
    unsigned char **out_sk_ptr, size_t *out_sk_len);

int tfhe_encrypt_u64(
    const unsigned char *ck_ptr, size_t ck_len,
    uint64_t plaintext,
    unsigned char **out_ct_ptr, size_t *out_ct_len);

int tfhe_decrypt_u64(
    const unsigned char *ck_ptr, size_t ck_len,
    const unsigned char *ct_ptr, size_t ct_len,
    uint64_t *out_plain);

int tfhe_add_u64(
    const unsigned char *sk_ptr, size_t sk_len,
    const unsigned char *a_ptr, size_t a_len,
    const unsigned char *b_ptr, size_t b_len,
    unsigned char **out_ct_ptr, size_t *out_ct_len);

void tfhe_free(unsigned char *ptr, size_t len);
*/
import "C"

import (
	"errors"
	"unsafe"
)

var (
	ErrNullPtr     = errors.New("tfhebridge: null pointer")
	ErrDeserialize = errors.New("tfhebridge: deserialize failure")
	ErrSerialize   = errors.New("tfhebridge: serialize failure")
	ErrOperation   = errors.New("tfhebridge: operation failure")
	ErrUnknown     = errors.New("tfhebridge: unknown error code")
)

func errFromCode(code C.int) error {
	switch code {
	case 0:
		return nil
	case 1:
		return ErrNullPtr
	case 2:
		return ErrDeserialize
	case 3:
		return ErrSerialize
	case 4:
		return ErrOperation
	default:
		return ErrUnknown
	}
}

// copyAndFree copies a Rust-allocated buffer into a Go slice and releases
// the underlying memory.
func copyAndFree(ptr *C.uchar, length C.size_t) []byte {
	if ptr == nil || length == 0 {
		return nil
	}
	out := C.GoBytes(unsafe.Pointer(ptr), C.int(length))
	C.tfhe_free(ptr, length)
	return out
}

// Keygen generates a fresh (clientKey, serverKey) pair. Both keys are
// serialised via bincode on the Rust side. The client key is the secret
// (decryption capability); the server key is public-evaluation only.
func Keygen() (clientKey, serverKey []byte, err error) {
	var ckPtr, skPtr *C.uchar
	var ckLen, skLen C.size_t
	code := C.tfhe_keygen(&ckPtr, &ckLen, &skPtr, &skLen)
	if e := errFromCode(code); e != nil {
		return nil, nil, e
	}
	return copyAndFree(ckPtr, ckLen), copyAndFree(skPtr, skLen), nil
}

// EncryptU64 produces a ciphertext of the given plaintext under clientKey.
func EncryptU64(clientKey []byte, plaintext uint64) ([]byte, error) {
	if len(clientKey) == 0 {
		return nil, ErrNullPtr
	}
	var ctPtr *C.uchar
	var ctLen C.size_t
	code := C.tfhe_encrypt_u64(
		(*C.uchar)(unsafe.Pointer(&clientKey[0])), C.size_t(len(clientKey)),
		C.uint64_t(plaintext),
		&ctPtr, &ctLen,
	)
	if e := errFromCode(code); e != nil {
		return nil, e
	}
	return copyAndFree(ctPtr, ctLen), nil
}

// DecryptU64 recovers the plaintext from a ciphertext using clientKey.
func DecryptU64(clientKey, ciphertext []byte) (uint64, error) {
	if len(clientKey) == 0 || len(ciphertext) == 0 {
		return 0, ErrNullPtr
	}
	var plain C.uint64_t
	code := C.tfhe_decrypt_u64(
		(*C.uchar)(unsafe.Pointer(&clientKey[0])), C.size_t(len(clientKey)),
		(*C.uchar)(unsafe.Pointer(&ciphertext[0])), C.size_t(len(ciphertext)),
		&plain,
	)
	if e := errFromCode(code); e != nil {
		return 0, e
	}
	return uint64(plain), nil
}

// AddU64 computes the homomorphic sum of two ciphertexts. The serverKey
// is required for the operation. No bootstrap is performed (tfhe-rs
// FheUint64 add is a non-bootstrap op).
func AddU64(serverKey, a, b []byte) ([]byte, error) {
	if len(serverKey) == 0 || len(a) == 0 || len(b) == 0 {
		return nil, ErrNullPtr
	}
	var ctPtr *C.uchar
	var ctLen C.size_t
	code := C.tfhe_add_u64(
		(*C.uchar)(unsafe.Pointer(&serverKey[0])), C.size_t(len(serverKey)),
		(*C.uchar)(unsafe.Pointer(&a[0])), C.size_t(len(a)),
		(*C.uchar)(unsafe.Pointer(&b[0])), C.size_t(len(b)),
		&ctPtr, &ctLen,
	)
	if e := errFromCode(code); e != nil {
		return nil, e
	}
	return copyAndFree(ctPtr, ctLen), nil
}
