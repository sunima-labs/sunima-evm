package tfhe

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sunima-labs/sunima-evm/x/tfhe/keeper"
	"github.com/sunima-labs/sunima-evm/x/tfhe/types"
)

// InitGenesis seeds module state from a GenesisState. Ciphertexts and
// attesters from genesis are written verbatim; params are validated
// upstream (AppModuleBasic.ValidateGenesis) before this runs.
//
// Stage 5.1: ciphertext write loop is the only state-touching path here.
// The 5-of-9 attester registry materialises in Stage 5.3.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, gs *types.GenesisState) {
	if gs == nil {
		return
	}
	// Install the genesis server key onto the keeper so HomomorphicCompute
	// has access to it from the very first block. Operators ship the key
	// inside genesis.json under app_state.tfhe.params.server_key.
	if len(gs.Params.ServerKey) > 0 {
		k.SetServerKey(gs.Params.ServerKey)
	}
	for i := range gs.Ciphertexts {
		ct := &gs.Ciphertexts[i]
		// Genesis ciphertexts bypass duplicate detection: if the same id
		// appears twice the second overwrite will surface during state
		// validation (gs.Validate).
		_, _ = k.StoreCiphertext(ctx, ct.Data, ct.Owner)
	}
}

// ExportGenesis returns the current module state for chain snapshot.
// Stage 5.1: ciphertext list export is deferred — iterating the
// ownership prefix range without a paged response would balloon the
// snapshot, and the chain has no consumers of the dump yet. Empty
// list keeps export shape stable for future stages.
func ExportGenesis(_ sdk.Context, _ keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Params:      types.DefaultParams(),
		Ciphertexts: nil,
		Attesters:   nil,
	}
}
