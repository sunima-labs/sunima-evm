// Package keeper holds the x/tfhe state-touching logic.
//
// Stage 5.1 Week 3 Phase 2: keeper now operates against a real
// Cosmos SDK store via sdk.Context.KVStore(storeKey). Unit tests
// drive it through testutil.DefaultContext(t, key, tkey) — the same
// API path the production chain uses.
package keeper

import (
	"crypto/sha256"
	"errors"

	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sunima-labs/sunima-evm/internal/tfhebridge"
	"github.com/sunima-labs/sunima-evm/x/tfhe/types"
)

// OpAdd is the only supported homomorphic operation in Stage 5.1.
// mul / compare require bootstrap and are deferred to Stage 5.4.
const OpAdd = "add"

// Keeper handles the x/tfhe module's state and dispatches homomorphic
// operations to the FFI bridge.
//
// Stage 5.3 will add an AttesterRegistry field; Stage 5.4 will move
// serverKey into params storage and reload it lazily.
type Keeper struct {
	cdc       codec.BinaryCodec
	storeKey  storetypes.StoreKey
	authority sdk.AccAddress
	serverKey []byte // tfhe-rs ServerKey blob; required for homomorphic ops
}

// NewKeeper constructs a keeper. authority is the gov module account
// for params updates and attester registration; serverKey may be nil for
// read-only paths — HomomorphicCompute returns ErrServerKeyNotSet if
// invoked without one.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	authority sdk.AccAddress,
	serverKey []byte,
) Keeper {
	if err := sdk.VerifyAddressFormat(authority); err != nil {
		panic(err)
	}
	return Keeper{cdc: cdc, storeKey: storeKey, authority: authority, serverKey: serverKey}
}

// Authority returns the configured authority address as a bech32 string,
// matching the format used in MsgUpdateParams / MsgRegisterAttester.
func (k Keeper) Authority() string { return k.authority.String() }

// ServerKey exposes the configured server key for diagnostics/tests.
func (k Keeper) ServerKey() []byte { return k.serverKey }

// StoreCiphertext persists ciphertext addressed by sha256(ciphertext).
// Returns the content-addressed id. If the same ciphertext is submitted
// twice (by hash) the call returns the existing id and ErrCiphertextAlreadyExists.
func (k Keeper) StoreCiphertext(ctx sdk.Context, ciphertext []byte, owner string) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, types.ErrEmptyCiphertext
	}
	if owner == "" {
		return nil, types.ErrEmptyOwner
	}
	store := ctx.KVStore(k.storeKey)
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
func (k Keeper) GetCiphertext(ctx sdk.Context, id []byte, caller string) ([]byte, error) {
	if len(id) == 0 {
		return nil, types.ErrCiphertextNotFound
	}
	store := ctx.KVStore(k.storeKey)
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
func (k Keeper) HomomorphicCompute(ctx sdk.Context, opType string, inputIDs [][]byte, caller string) ([]byte, error) {
	if opType != OpAdd {
		return nil, types.ErrInvalidOpType
	}
	if len(k.serverKey) == 0 {
		return nil, types.ErrServerKeyNotSet
	}
	if len(inputIDs) != 2 {
		return nil, types.ErrInvalidInputCount
	}
	ctA, err := k.GetCiphertext(ctx, inputIDs[0], caller)
	if err != nil {
		return nil, err
	}
	ctB, err := k.GetCiphertext(ctx, inputIDs[1], caller)
	if err != nil {
		return nil, err
	}
	result, err := tfhebridge.AddU64(k.serverKey, ctA, ctB)
	if err != nil {
		return nil, errors.Join(types.ErrInvalidCiphertext, err)
	}
	resultID, err := k.StoreCiphertext(ctx, result, caller)
	if err != nil && !errors.Is(err, types.ErrCiphertextAlreadyExists) {
		return nil, err
	}
	return resultID, nil
}

// VerifyAttestationQuorum is unchanged scaffold — implementation lands in
// Stage 5.3 alongside the threshold-decryption flow.
func (k Keeper) VerifyAttestationQuorum(_ sdk.Context, _ []byte, _ [][]byte, _ [][]byte) error {
	// TODO Stage 5.3: lookup registered attesters, verify each signature,
	// require >=5 distinct, then combine partials via tfhebridge.
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
