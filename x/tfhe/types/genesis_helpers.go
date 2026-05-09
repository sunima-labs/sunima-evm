package types

import (
	"bytes"
	"fmt"
)

// DefaultParams returns a baseline Params suitable for a fresh chain.
// server_key is intentionally empty — the chain operator must inject
// the bincode-serialised tfhe-rs server key via genesis.json (or a
// post-launch governance MsgUpdateParams once Stage 5.1 ships) before
// HomomorphicCompute can succeed.
func DefaultParams() Params {
	return Params{
		MinAttesters:           5,
		DecryptionWindowBlocks: 1000,
		ServerKey:              nil,
	}
}

// DefaultGenesisState returns the default x/tfhe genesis state — empty
// ciphertext list, empty attester list, default params.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:      DefaultParams(),
		Ciphertexts: nil,
		Attesters:   nil,
	}
}

// Validate runs basic sanity checks on the genesis state. Heavy checks
// (e.g. ciphertext deserialisation) are deferred to runtime to keep
// chain start-up fast.
func (gs *GenesisState) Validate() error {
	if gs == nil {
		return fmt.Errorf("genesis state is nil")
	}
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	seen := make(map[string]struct{}, len(gs.Ciphertexts))
	for i := range gs.Ciphertexts {
		ct := &gs.Ciphertexts[i]
		if len(ct.Id) != 32 {
			return fmt.Errorf("ciphertext %d: id must be 32 bytes, got %d", i, len(ct.Id))
		}
		if len(ct.Data) == 0 {
			return fmt.Errorf("ciphertext %d: data is empty", i)
		}
		if ct.Owner == "" {
			return fmt.Errorf("ciphertext %d: owner is empty", i)
		}
		key := string(ct.Id)
		if _, dup := seen[key]; dup {
			return fmt.Errorf("ciphertext %d: duplicate id", i)
		}
		seen[key] = struct{}{}
	}
	for i := range gs.Attesters {
		a := &gs.Attesters[i]
		if a.Address == "" {
			return fmt.Errorf("attester %d: address is empty", i)
		}
		if len(a.Pubkey) == 0 {
			return fmt.Errorf("attester %d: pubkey is empty", i)
		}
	}
	return nil
}

// Validate runs lightweight sanity checks on Params.
func (p Params) Validate() error {
	if p.MinAttesters == 0 {
		return fmt.Errorf("min_attesters must be > 0")
	}
	if p.DecryptionWindowBlocks == 0 {
		return fmt.Errorf("decryption_window_blocks must be > 0")
	}
	// server_key may be empty at chain start; HomomorphicCompute returns
	// ErrServerKeyNotSet until a real key is loaded via UpdateParams.
	if len(p.ServerKey) > 0 && len(p.ServerKey) < 64 {
		return fmt.Errorf("server_key looks too small (%d bytes) — expected >=64", len(p.ServerKey))
	}
	// Avoid pointlessly comparing against bytes.NewReader allocations on every
	// validate call — keep it cheap.
	_ = bytes.Equal
	return nil
}
