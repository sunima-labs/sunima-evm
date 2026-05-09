package keeper_test

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"testing"

	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/testutil"

	"github.com/sunima-labs/sunima-evm/internal/tfhebridge"
	"github.com/sunima-labs/sunima-evm/x/tfhe/keeper"
	"github.com/sunima-labs/sunima-evm/x/tfhe/types"
)

// ──────────────────────────────────────────────────────────────────────────
// Shared TFHE keys — keygen costs ~870ms on Netcup x86_64, so we generate
// once per test binary instead of per test case.
// ──────────────────────────────────────────────────────────────────────────

var (
	keygenOnce sync.Once
	clientKey  []byte
	serverKey  []byte
	keygenErr  error
)

func sharedKeys(t *testing.T) (ck, sk []byte) {
	t.Helper()
	keygenOnce.Do(func() {
		clientKey, serverKey, keygenErr = tfhebridge.Keygen()
	})
	if keygenErr != nil {
		t.Fatalf("keygen failed: %v", keygenErr)
	}
	return clientKey, serverKey
}

func mustEncrypt(t *testing.T, ck []byte, v uint64) []byte {
	t.Helper()
	ct, err := tfhebridge.EncryptU64(ck, v)
	if err != nil {
		t.Fatalf("encrypt(%d) failed: %v", v, err)
	}
	return ct
}

func mustDecrypt(t *testing.T, ck, ct []byte) uint64 {
	t.Helper()
	v, err := tfhebridge.DecryptU64(ck, ct)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	return v
}

// setupKeeper builds a fresh in-memory keeper and a clean sdk.Context
// for each test. Mirrors the cosmos-evm/x/vm pattern (testutil
// .DefaultContext + storetypes.NewKVStoreKey).
func setupKeeper(t *testing.T, withServerKey bool) (keeper.Keeper, sdk.Context) {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey("transient_test")
	testCtx := testutil.DefaultContext(key, tkey)

	encCfg := moduletestutil.MakeTestEncodingConfig()
	authority := sdk.AccAddress("tfhe-authority-test-")

	var sk []byte
	if withServerKey {
		_, sk = sharedKeys(t)
	}
	k := keeper.NewKeeper(encCfg.Codec, key, authority, sk)
	return k, testCtx
}

// ──────────────────────────────────────────────────────────────────────────
// StoreCiphertext / GetCiphertext
// ──────────────────────────────────────────────────────────────────────────

func TestStoreCiphertext_HappyPath(t *testing.T) {
	ck, _ := sharedKeys(t)
	k, ctx := setupKeeper(t, true)
	owner := "sunima1alice"

	for i := uint64(1); i <= 10; i++ {
		ct := mustEncrypt(t, ck, i)
		id, err := k.StoreCiphertext(ctx, ct, owner)
		if err != nil {
			t.Fatalf("StoreCiphertext(i=%d): %v", i, err)
		}
		if len(id) != 32 {
			t.Fatalf("expected 32-byte sha256 id, got %d bytes", len(id))
		}

		got, err := k.GetCiphertext(ctx, id, owner)
		if err != nil {
			t.Fatalf("GetCiphertext(i=%d): %v", i, err)
		}
		if !bytes.Equal(got, ct) {
			t.Fatalf("round-trip mismatch at i=%d", i)
		}
	}
}

func TestStoreCiphertext_EmptyInputs(t *testing.T) {
	k, ctx := setupKeeper(t, true)

	if _, err := k.StoreCiphertext(ctx, nil, "sunima1alice"); !errors.Is(err, types.ErrEmptyCiphertext) {
		t.Fatalf("nil ct: want ErrEmptyCiphertext, got %v", err)
	}
	if _, err := k.StoreCiphertext(ctx, []byte{0x01}, ""); !errors.Is(err, types.ErrEmptyOwner) {
		t.Fatalf("empty owner: want ErrEmptyOwner, got %v", err)
	}
}

func TestStoreCiphertext_DuplicateRejected(t *testing.T) {
	ck, _ := sharedKeys(t)
	k, ctx := setupKeeper(t, true)
	owner := "sunima1alice"
	ct := mustEncrypt(t, ck, 42)

	id1, err := k.StoreCiphertext(ctx, ct, owner)
	if err != nil {
		t.Fatalf("first store: %v", err)
	}
	id2, err := k.StoreCiphertext(ctx, ct, owner)
	if !errors.Is(err, types.ErrCiphertextAlreadyExists) {
		t.Fatalf("second store: want ErrCiphertextAlreadyExists, got %v", err)
	}
	if !bytes.Equal(id1, id2) {
		t.Fatalf("content-addressed id should be stable across duplicate stores")
	}
}

func TestGetCiphertext_NonOwnerRejected(t *testing.T) {
	ck, _ := sharedKeys(t)
	k, ctx := setupKeeper(t, true)
	alice, bob := "sunima1alice", "sunima1bob"

	ct := mustEncrypt(t, ck, 7)
	id, err := k.StoreCiphertext(ctx, ct, alice)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if _, err := k.GetCiphertext(ctx, id, bob); !errors.Is(err, types.ErrUnauthorized) {
		t.Fatalf("bob should not read alice's ct, got err=%v", err)
	}
	if _, err := k.GetCiphertext(ctx, id, ""); !errors.Is(err, types.ErrUnauthorized) {
		t.Fatalf("empty caller should be rejected, got err=%v", err)
	}
}

func TestGetCiphertext_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t, true)
	missing := bytes.Repeat([]byte{0xab}, 32)
	if _, err := k.GetCiphertext(ctx, missing, "sunima1alice"); !errors.Is(err, types.ErrCiphertextNotFound) {
		t.Fatalf("want ErrCiphertextNotFound, got %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────
// HomomorphicCompute (OpAdd) — the headline E2E test of Week 2.
// ──────────────────────────────────────────────────────────────────────────

func TestHomomorphicCompute_AddE2E(t *testing.T) {
	ck, _ := sharedKeys(t)
	k, ctx := setupKeeper(t, true)
	owner := "sunima1alice"

	ctA := mustEncrypt(t, ck, 5)
	ctB := mustEncrypt(t, ck, 7)
	idA, err := k.StoreCiphertext(ctx, ctA, owner)
	if err != nil {
		t.Fatalf("store A: %v", err)
	}
	idB, err := k.StoreCiphertext(ctx, ctB, owner)
	if err != nil {
		t.Fatalf("store B: %v", err)
	}

	resultID, err := k.HomomorphicCompute(ctx, keeper.OpAdd, [][]byte{idA, idB}, owner)
	if err != nil {
		t.Fatalf("HomomorphicCompute: %v", err)
	}

	resultCT, err := k.GetCiphertext(ctx, resultID, owner)
	if err != nil {
		t.Fatalf("get result: %v", err)
	}
	got := mustDecrypt(t, ck, resultCT)
	if got != 12 {
		t.Fatalf("homomorphic 5+7: got %d, want 12", got)
	}
}

func TestHomomorphicCompute_Variations(t *testing.T) {
	ck, _ := sharedKeys(t)
	k, ctx := setupKeeper(t, true)
	owner := "sunima1alice"

	cases := []struct{ a, b, want uint64 }{
		{0, 0, 0},
		{1, 0, 1},
		{1000, 234, 1234},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d_plus_%d", tc.a, tc.b), func(t *testing.T) {
			idA, err := k.StoreCiphertext(ctx, mustEncrypt(t, ck, tc.a), owner)
			if err != nil && !errors.Is(err, types.ErrCiphertextAlreadyExists) {
				t.Fatalf("store a: %v", err)
			}
			idB, err := k.StoreCiphertext(ctx, mustEncrypt(t, ck, tc.b), owner)
			if err != nil && !errors.Is(err, types.ErrCiphertextAlreadyExists) {
				t.Fatalf("store b: %v", err)
			}
			resID, err := k.HomomorphicCompute(ctx, keeper.OpAdd, [][]byte{idA, idB}, owner)
			if err != nil {
				t.Fatalf("compute: %v", err)
			}
			resCT, err := k.GetCiphertext(ctx, resID, owner)
			if err != nil {
				t.Fatalf("get result: %v", err)
			}
			got := mustDecrypt(t, ck, resCT)
			if got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestHomomorphicCompute_NonOwnerRejected(t *testing.T) {
	ck, _ := sharedKeys(t)
	k, ctx := setupKeeper(t, true)
	alice, bob := "sunima1alice", "sunima1bob"

	idA, err := k.StoreCiphertext(ctx, mustEncrypt(t, ck, 9), alice)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	idB, err := k.StoreCiphertext(ctx, mustEncrypt(t, ck, 11), alice)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if _, err := k.HomomorphicCompute(ctx, keeper.OpAdd, [][]byte{idA, idB}, bob); !errors.Is(err, types.ErrUnauthorized) {
		t.Fatalf("bob computing on alice's inputs: want ErrUnauthorized, got %v", err)
	}
}

func TestHomomorphicCompute_InvalidInputs(t *testing.T) {
	ck, _ := sharedKeys(t)
	k, ctx := setupKeeper(t, true)
	owner := "sunima1alice"

	idA, err := k.StoreCiphertext(ctx, mustEncrypt(t, ck, 1), owner)
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	if _, err := k.HomomorphicCompute(ctx, "mul", [][]byte{idA, idA}, owner); !errors.Is(err, types.ErrInvalidOpType) {
		t.Fatalf("unknown op: want ErrInvalidOpType, got %v", err)
	}
	if _, err := k.HomomorphicCompute(ctx, keeper.OpAdd, [][]byte{idA}, owner); !errors.Is(err, types.ErrInvalidInputCount) {
		t.Fatalf("1 input: want ErrInvalidInputCount, got %v", err)
	}
	if _, err := k.HomomorphicCompute(ctx, keeper.OpAdd, [][]byte{idA, idA, idA}, owner); !errors.Is(err, types.ErrInvalidInputCount) {
		t.Fatalf("3 inputs: want ErrInvalidInputCount, got %v", err)
	}
}

func TestHomomorphicCompute_NoServerKey(t *testing.T) {
	ck, _ := sharedKeys(t)
	k, ctx := setupKeeper(t, false) // explicitly no server key
	owner := "sunima1alice"

	idA, err := k.StoreCiphertext(ctx, mustEncrypt(t, ck, 1), owner)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	idB, err := k.StoreCiphertext(ctx, mustEncrypt(t, ck, 2), owner)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if _, err := k.HomomorphicCompute(ctx, keeper.OpAdd, [][]byte{idA, idB}, owner); !errors.Is(err, types.ErrServerKeyNotSet) {
		t.Fatalf("want ErrServerKeyNotSet, got %v", err)
	}
}
