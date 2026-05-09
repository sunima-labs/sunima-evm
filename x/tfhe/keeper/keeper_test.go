package keeper_test

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"testing"

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

// ──────────────────────────────────────────────────────────────────────────
// MemKVStore satisfies KVStore (compile-time check).
// ──────────────────────────────────────────────────────────────────────────

var _ keeper.KVStore = (*keeper.MemKVStore)(nil)

// ──────────────────────────────────────────────────────────────────────────
// StoreCiphertext / GetCiphertext
// ──────────────────────────────────────────────────────────────────────────

func TestStoreCiphertext_HappyPath(t *testing.T) {
	ck, sk := sharedKeys(t)
	k := keeper.NewKeeper(sk)
	store := keeper.NewMemKVStore()
	owner := "sunima1alice"

	// store 10 distinct ciphertexts, each must round-trip via the keeper.
	for i := uint64(1); i <= 10; i++ {
		ct := mustEncrypt(t, ck, i)
		id, err := k.StoreCiphertext(store, ct, owner)
		if err != nil {
			t.Fatalf("StoreCiphertext(i=%d): %v", i, err)
		}
		if len(id) != 32 {
			t.Fatalf("expected 32-byte sha256 id, got %d bytes", len(id))
		}

		got, err := k.GetCiphertext(store, id, owner)
		if err != nil {
			t.Fatalf("GetCiphertext(i=%d): %v", i, err)
		}
		if !bytes.Equal(got, ct) {
			t.Fatalf("round-trip mismatch at i=%d", i)
		}
	}
}

func TestStoreCiphertext_EmptyInputs(t *testing.T) {
	_, sk := sharedKeys(t)
	k := keeper.NewKeeper(sk)
	store := keeper.NewMemKVStore()

	if _, err := k.StoreCiphertext(store, nil, "sunima1alice"); !errors.Is(err, types.ErrEmptyCiphertext) {
		t.Fatalf("nil ct: want ErrEmptyCiphertext, got %v", err)
	}
	if _, err := k.StoreCiphertext(store, []byte{0x01}, ""); !errors.Is(err, types.ErrEmptyOwner) {
		t.Fatalf("empty owner: want ErrEmptyOwner, got %v", err)
	}
}

func TestStoreCiphertext_DuplicateRejected(t *testing.T) {
	ck, sk := sharedKeys(t)
	k := keeper.NewKeeper(sk)
	store := keeper.NewMemKVStore()
	owner := "sunima1alice"
	ct := mustEncrypt(t, ck, 42)

	id1, err := k.StoreCiphertext(store, ct, owner)
	if err != nil {
		t.Fatalf("first store: %v", err)
	}
	id2, err := k.StoreCiphertext(store, ct, owner)
	if !errors.Is(err, types.ErrCiphertextAlreadyExists) {
		t.Fatalf("second store: want ErrCiphertextAlreadyExists, got %v", err)
	}
	if !bytes.Equal(id1, id2) {
		t.Fatalf("content-addressed id should be stable across duplicate stores")
	}
}

func TestGetCiphertext_NonOwnerRejected(t *testing.T) {
	ck, sk := sharedKeys(t)
	k := keeper.NewKeeper(sk)
	store := keeper.NewMemKVStore()
	alice, bob := "sunima1alice", "sunima1bob"

	ct := mustEncrypt(t, ck, 7)
	id, err := k.StoreCiphertext(store, ct, alice)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if _, err := k.GetCiphertext(store, id, bob); !errors.Is(err, types.ErrUnauthorized) {
		t.Fatalf("bob should not read alice's ct, got err=%v", err)
	}
	if _, err := k.GetCiphertext(store, id, ""); !errors.Is(err, types.ErrUnauthorized) {
		t.Fatalf("empty caller should be rejected, got err=%v", err)
	}
}

func TestGetCiphertext_NotFound(t *testing.T) {
	_, sk := sharedKeys(t)
	k := keeper.NewKeeper(sk)
	store := keeper.NewMemKVStore()

	missing := bytes.Repeat([]byte{0xab}, 32)
	if _, err := k.GetCiphertext(store, missing, "sunima1alice"); !errors.Is(err, types.ErrCiphertextNotFound) {
		t.Fatalf("want ErrCiphertextNotFound, got %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────
// HomomorphicCompute (OpAdd) — the headline E2E test of Week 2.
// ──────────────────────────────────────────────────────────────────────────

func TestHomomorphicCompute_AddE2E(t *testing.T) {
	ck, sk := sharedKeys(t)
	k := keeper.NewKeeper(sk)
	store := keeper.NewMemKVStore()
	owner := "sunima1alice"

	// 5 + 7 → 12, decrypted via the keeper-managed ciphertext.
	ctA := mustEncrypt(t, ck, 5)
	ctB := mustEncrypt(t, ck, 7)
	idA, err := k.StoreCiphertext(store, ctA, owner)
	if err != nil {
		t.Fatalf("store A: %v", err)
	}
	idB, err := k.StoreCiphertext(store, ctB, owner)
	if err != nil {
		t.Fatalf("store B: %v", err)
	}

	resultID, err := k.HomomorphicCompute(store, keeper.OpAdd, [][]byte{idA, idB}, owner)
	if err != nil {
		t.Fatalf("HomomorphicCompute: %v", err)
	}

	resultCT, err := k.GetCiphertext(store, resultID, owner)
	if err != nil {
		t.Fatalf("get result: %v", err)
	}
	got := mustDecrypt(t, ck, resultCT)
	if got != 12 {
		t.Fatalf("homomorphic 5+7: got %d, want 12", got)
	}
}

func TestHomomorphicCompute_Variations(t *testing.T) {
	ck, sk := sharedKeys(t)
	k := keeper.NewKeeper(sk)
	store := keeper.NewMemKVStore()
	owner := "sunima1alice"

	// Smaller table to keep keygen overhead bounded — cases use the shared
	// server key but each encrypts fresh ciphertexts.
	cases := []struct{ a, b, want uint64 }{
		{0, 0, 0},
		{1, 0, 1},
		{1000, 234, 1234},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d_plus_%d", tc.a, tc.b), func(t *testing.T) {
			idA, err := k.StoreCiphertext(store, mustEncrypt(t, ck, tc.a), owner)
			if err != nil && !errors.Is(err, types.ErrCiphertextAlreadyExists) {
				t.Fatalf("store a: %v", err)
			}
			idB, err := k.StoreCiphertext(store, mustEncrypt(t, ck, tc.b), owner)
			if err != nil && !errors.Is(err, types.ErrCiphertextAlreadyExists) {
				t.Fatalf("store b: %v", err)
			}
			resID, err := k.HomomorphicCompute(store, keeper.OpAdd, [][]byte{idA, idB}, owner)
			if err != nil {
				t.Fatalf("compute: %v", err)
			}
			resCT, err := k.GetCiphertext(store, resID, owner)
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
	ck, sk := sharedKeys(t)
	k := keeper.NewKeeper(sk)
	store := keeper.NewMemKVStore()
	alice, bob := "sunima1alice", "sunima1bob"

	idA, err := k.StoreCiphertext(store, mustEncrypt(t, ck, 9), alice)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	idB, err := k.StoreCiphertext(store, mustEncrypt(t, ck, 11), alice)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if _, err := k.HomomorphicCompute(store, keeper.OpAdd, [][]byte{idA, idB}, bob); !errors.Is(err, types.ErrUnauthorized) {
		t.Fatalf("bob computing on alice's inputs: want ErrUnauthorized, got %v", err)
	}
}

func TestHomomorphicCompute_InvalidInputs(t *testing.T) {
	ck, sk := sharedKeys(t)
	k := keeper.NewKeeper(sk)
	store := keeper.NewMemKVStore()
	owner := "sunima1alice"

	idA, err := k.StoreCiphertext(store, mustEncrypt(t, ck, 1), owner)
	if err != nil {
		t.Fatalf("store: %v", err)
	}

	// unknown op
	if _, err := k.HomomorphicCompute(store, "mul", [][]byte{idA, idA}, owner); !errors.Is(err, types.ErrInvalidOpType) {
		t.Fatalf("unknown op: want ErrInvalidOpType, got %v", err)
	}

	// wrong input count (1, not 2)
	if _, err := k.HomomorphicCompute(store, keeper.OpAdd, [][]byte{idA}, owner); !errors.Is(err, types.ErrInvalidInputCount) {
		t.Fatalf("1 input: want ErrInvalidInputCount, got %v", err)
	}

	// 3 inputs
	if _, err := k.HomomorphicCompute(store, keeper.OpAdd, [][]byte{idA, idA, idA}, owner); !errors.Is(err, types.ErrInvalidInputCount) {
		t.Fatalf("3 inputs: want ErrInvalidInputCount, got %v", err)
	}
}

func TestHomomorphicCompute_NoServerKey(t *testing.T) {
	ck, _ := sharedKeys(t)
	k := keeper.NewKeeper(nil) // explicitly no server key
	store := keeper.NewMemKVStore()
	owner := "sunima1alice"

	idA, err := k.StoreCiphertext(store, mustEncrypt(t, ck, 1), owner)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	idB, err := k.StoreCiphertext(store, mustEncrypt(t, ck, 2), owner)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if _, err := k.HomomorphicCompute(store, keeper.OpAdd, [][]byte{idA, idB}, owner); !errors.Is(err, types.ErrServerKeyNotSet) {
		t.Fatalf("want ErrServerKeyNotSet, got %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────
// MemKVStore basics — sanity checks on the test-double itself.
// ──────────────────────────────────────────────────────────────────────────

func TestMemKVStore_BasicOps(t *testing.T) {
	s := keeper.NewMemKVStore()
	k1, v1 := []byte{0x01, 0xaa}, []byte("hello")
	k2, v2 := []byte{0x01, 0xbb}, []byte("world")

	if s.Has(k1) {
		t.Fatal("empty store should not Has(k1)")
	}
	s.Set(k1, v1)
	s.Set(k2, v2)

	if got := s.Get(k1); !bytes.Equal(got, v1) {
		t.Fatalf("Get(k1): %q vs %q", got, v1)
	}
	if !s.Has(k2) {
		t.Fatal("Has(k2) after Set should be true")
	}

	// Mutating the value passed to Set must not affect what's stored.
	v1[0] = 'X'
	if got := s.Get(k1); got[0] == 'X' {
		t.Fatal("Set must copy value, not retain caller's slice")
	}

	s.Delete(k1)
	if s.Has(k1) || s.Get(k1) != nil {
		t.Fatal("Delete(k1) failed")
	}
}

func TestMemKVStore_IteratorRange(t *testing.T) {
	s := keeper.NewMemKVStore()
	s.Set([]byte("a"), []byte("1"))
	s.Set([]byte("b"), []byte("2"))
	s.Set([]byte("c"), []byte("3"))
	s.Set([]byte("d"), []byte("4"))

	it := s.Iterator([]byte("b"), []byte("d")) // half-open [b, d) → b, c
	defer it.Close()
	keys := []string{}
	for ; it.Valid(); it.Next() {
		keys = append(keys, string(it.Key()))
	}
	want := []string{"b", "c"}
	if len(keys) != len(want) {
		t.Fatalf("iterator: got %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("iterator: got %v, want %v", keys, want)
		}
	}
}
