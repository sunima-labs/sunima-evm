package tfhe

import (
	"github.com/sunima-labs/sunima-evm/x/tfhe/keeper"
)

// GenesisState is the chain genesis representation for this module.
type GenesisState struct {
	Attesters []AttesterGenesisEntry
	// Params Params
}

// AttesterGenesisEntry seeds an initial attester at chain genesis.
type AttesterGenesisEntry struct {
	Address     string
	PubKey      []byte
	SharePubkey []byte
}

// DefaultGenesis returns an empty genesis state — chain operator must populate
// initial attesters via genesis.json or via governance after launch.
func DefaultGenesis() GenesisState {
	return GenesisState{Attesters: nil}
}

// InitGenesis seeds the keeper from a GenesisState.
func InitGenesis(k keeper.Keeper, gs GenesisState) {
	// TODO: persist each attester, set params
	_ = k
	_ = gs
}

// ExportGenesis dumps current module state for chain snapshot.
func ExportGenesis(k keeper.Keeper) GenesisState {
	// TODO: enumerate attesters, dump params
	_ = k
	return GenesisState{}
}
