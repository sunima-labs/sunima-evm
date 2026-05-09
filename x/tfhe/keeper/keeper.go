// Package keeper holds the x/tfhe state-touching logic.
//
// Stage 5.1 Week 2 (this file): keeper operates against the local
// KVStore interface in store.go and delegates homomorphic ops to
// internal/tfhebridge. No Cosmos SDK runtime dependency yet — that
// arrives in Week 3 along with proto codegen and module wiring.
package keeper

import (
	"crypto/sha256"
	"errors"

	"github.com/sunima-labs/sunima-evm/internal/tfhebridge"
	"github.com/sunima-labs/sunima-evm/x/tfhe/types"
)

// OpAdd is the only supported homomorphic operation in Stage 5.1.
// mul / compare require bootstrap and are deferred to Stage 5.4.
const OpAdd = "add"

// Keeper handles the x/tfhe module's state and dispatches homomorphic
// operations to the FFI bridge.
//
// In Week 3 this struct gains storeKey + cdc + authority + AttesterRegistry
// fields once Cosmos SDK is wired. For now it carries the FHE server key
// directly — production storage layout (genesis import / governance update)
// is decided in Week 3.
type Keeper struct {
	serverKey []byte // tfhe-rs ServerKey blob; required for homomorphic ops
}

// NewKeeper constructs a keeper. serverKey may be nil for read-only paths
// (StoreCiphertext / GetCiphertext) — HomomorphicCompute returns
// ErrServerKeyNotSet if invoked without one.
func NewKeeper(serverKey []byte) Keeper {
	return Keeper{serverKey: serverKey}
}

// ServerKey exposes the configured server key for diagnostics/tests.
func (k Keeper) ServerKey() []byte { return k.serverKey }

// StoreCiphertext persists ciphertext addressed by sha256(ciphertext).
// Returns the content-addressed id. If the same ciphertext is submitted
// twice (by hash) the call returns ErrCiphertextAlreadyExists; the
// caller can decide whether to surface or swallow that.
func (k Keeper) StoreCiphertext(store KVStore, ciphertext []byte, owner string) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, types.ErrEmptyCiphertext
	}
	if owner == "" {
		return nil, types.ErrEmptyOwner
	}
	id := contentID(ciphertext)
	ctKey := ciphertextStoreKey(id)
	if store.Has(ctKey) {
		return id, types.ErrCiphertextAlreadyExists
	}
	store.Set(ctKey, ciphertext)
	store.Set(ownershipKey(owner, id), []byte{0x01})
	return id, nil
}

// GetCiphertext returns the stored ciphertext if `caller` owns it.
// Non-owners receive ErrUnauthorized; missing rows receive
// ErrCiphertextNotFound.
func (k Keeper) GetCiphertext(store KVStore, id []byte, caller string) ([]byte, error) {
	if len(id) == 0 {
		return nil, types.ErrCiphertextNotFound
	}
	ct := store.Get(ciphertextStoreKey(id))
	if ct == nil {
		return nil, types.ErrCiphertextNotFound
	}
	if caller == "" || !store.Has(ownershipKey(caller, id)) {
		return nil, types.ErrUnauthorized
	}
	return ct, nil
}

// HomomorphicCompute applies a registered op (Stage 5.1: only OpAdd) to
// the ciphertexts referenced by inputIDs. The caller must own every
// input. The result is stored owned by `caller` and its id returned.
//
// For OpAdd: exactly two inputs, sums them via tfhebridge.AddU64.
func (k Keeper) HomomorphicCompute(store KVStore, opType string, inputIDs [][]byte, caller string) ([]byte, error) {
	if opType != OpAdd {
		return nil, types.ErrInvalidOpType
	}
	if len(k.serverKey) == 0 {
		return nil, types.ErrServerKeyNotSet
	}
	if len(inputIDs) != 2 {
		return nil, types.ErrInvalidInputCount
	}
	ctA, err := k.GetCiphertext(store, inputIDs[0], caller)
	if err != nil {
		return nil, err
	}
	ctB, err := k.GetCiphertext(store, inputIDs[1], caller)
	if err != nil {
		return nil, err
	}
	result, err := tfhebridge.AddU64(k.serverKey, ctA, ctB)
	if err != nil {
		return nil, errors.Join(types.ErrInvalidCiphertext, err)
	}
	resultID, err := k.StoreCiphertext(store, result, caller)
	if err != nil && !errors.Is(err, types.ErrCiphertextAlreadyExists) {
		return nil, err
	}
	return resultID, nil
}

// VerifyAttestationQuorum is unchanged scaffold — implementation lands in
// Stage 5.3 alongside the threshold-decryption flow.
func (k Keeper) VerifyAttestationQuorum(_ []byte, _ [][]byte, _ [][]byte) error {
	// TODO Stage 5.3: lookup registered attesters, verify each signature,
	// require ≥5 distinct, then combine partials via tfhebridge.
	return nil
}

// ──────────────────────────────────────────────────────────────────────────
// Key derivation helpers — kept private to the keeper to ensure all
// callers go through the same prefix discipline.
// ──────────────────────────────────────────────────────────────────────────

func contentID(ct []byte) []byte {
	sum := sha256.Sum256(ct)
	return sum[:]
}

func ciphertextStoreKey(id []byte) []byte {
	out := make([]byte, 0, 1+len(id))
	out = append(out, types.CiphertextKeyPrefix...)
	out = append(out, id...)
	return out
}

func ownershipKey(owner string, id []byte) []byte {
	ownerB := []byte(owner)
	out := make([]byte, 0, 1+len(ownerB)+1+len(id))
	out = append(out, types.OwnershipIndexKeyPrefix...)
	out = append(out, ownerB...)
	out = append(out, 0x00) // separator to avoid prefix collisions on owner
	out = append(out, id...)
	return out
}
