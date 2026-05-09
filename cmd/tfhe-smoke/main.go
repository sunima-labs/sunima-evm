// tfhe-smoke is the Stage 5.1 Week 1 hello-world test.
//
// Validates the Rust↔Go FFI bridge end-to-end: keygen → encrypt 5 →
// encrypt 7 → homomorphic add → decrypt → assert == 12.
//
// Run:
//
//	nix develop --command bash -c 'cd internal/tfhebridge/rust && cargo build --release'
//	nix develop --command go run ./cmd/tfhe-smoke
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/sunima-labs/sunima-evm/internal/tfhebridge"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	const (
		valA     uint64 = 5
		valB     uint64 = 7
		expected uint64 = 12
	)

	t0 := time.Now()
	clientKey, serverKey, err := tfhebridge.Keygen()
	if err != nil {
		return fmt.Errorf("keygen: %w", err)
	}
	fmt.Printf("  keygen     %6d ms   ck=%d B  sk=%d B\n",
		time.Since(t0).Milliseconds(), len(clientKey), len(serverKey))

	t1 := time.Now()
	ctA, err := tfhebridge.EncryptU64(clientKey, valA)
	if err != nil {
		return fmt.Errorf("encrypt A: %w", err)
	}
	ctB, err := tfhebridge.EncryptU64(clientKey, valB)
	if err != nil {
		return fmt.Errorf("encrypt B: %w", err)
	}
	fmt.Printf("  encrypt x2 %6d ms   ctA=%d B  ctB=%d B\n",
		time.Since(t1).Milliseconds(), len(ctA), len(ctB))

	t2 := time.Now()
	ctSum, err := tfhebridge.AddU64(serverKey, ctA, ctB)
	if err != nil {
		return fmt.Errorf("add: %w", err)
	}
	fmt.Printf("  add        %6d ms   ctSum=%d B\n",
		time.Since(t2).Milliseconds(), len(ctSum))

	t3 := time.Now()
	got, err := tfhebridge.DecryptU64(clientKey, ctSum)
	if err != nil {
		return fmt.Errorf("decrypt: %w", err)
	}
	fmt.Printf("  decrypt    %6d ms   got=%d\n",
		time.Since(t3).Milliseconds(), got)

	if got != expected {
		return fmt.Errorf("got %d, want %d", got, expected)
	}

	// Also surface the ciphertext hash so the determinism check (Week 4
	// preview) has a stable artifact to compare across runs.
	sum := sha256.Sum256(ctSum)
	fmt.Printf("  ctSum sha256=%s\n", hex.EncodeToString(sum[:]))
	fmt.Printf("PASS  %d + %d = %d (homomorphic, total %d ms)\n",
		valA, valB, got, time.Since(t0).Milliseconds())
	return nil
}
