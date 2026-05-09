package tfhe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/sunima-labs/sunima-evm/x/tfhe/keeper"
	"github.com/sunima-labs/sunima-evm/x/tfhe/types"
)

// consensusVersion defines the current x/tfhe module consensus version.
const consensusVersion = 1

var (
	_ module.AppModule      = AppModule{} //nolint:staticcheck // keep for legacy purposes
	_ module.AppModuleBasic = AppModuleBasic{}

	_ module.HasGenesisBasics = AppModuleBasic{}
	_ appmodule.AppModule     = AppModule{}
)

// AppModuleBasic defines the basic application module used by the x/tfhe module.
type AppModuleBasic struct{}

// Name returns the x/tfhe module's name.
func (AppModuleBasic) Name() string { return types.ModuleName }

// RegisterLegacyAminoCodec registers Msg types for amino-aware clients.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// ConsensusVersion returns the consensus state-breaking version for the module.
func (AppModuleBasic) ConsensusVersion() uint64 { return consensusVersion }

// DefaultGenesis returns default genesis state as raw bytes for the
// x/tfhe module (empty ciphertext + attester lists, default params).
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis runs lightweight checks on the genesis JSON.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis: %w", types.ModuleName, err)
	}
	return gs.Validate()
}

// RegisterGRPCGatewayRoutes registers REST routes. Stage 5.1 ships
// without a generated grpc-gateway — the proto definitions carry
// google.api.http annotations but the corresponding .pb.gw.go was not
// produced during codegen. REST surface lights up in a follow-up
// alongside the OwnedCiphertexts pagination work.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {
}

// RegisterInterfaces registers x/tfhe interfaces and Msg implementations.
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// GetTxCmd returns the root tx command for the module. Stage 5.1 ships
// without dedicated CLI commands — chain operators interact via the
// generic `sunimad tx` Msg encoder. CLI helpers land in Phase 2 step 6.
func (AppModuleBasic) GetTxCmd() *cobra.Command { return nil }

// GetQueryCmd returns the root query command. See GetTxCmd note.
func (AppModuleBasic) GetQueryCmd() *cobra.Command { return nil }

// ____________________________________________________________________________

// AppModule implements an application module for the x/tfhe module.
type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
}

// NewAppModule creates a new AppModule object.
func NewAppModule(k keeper.Keeper) AppModule {
	return AppModule{AppModuleBasic: AppModuleBasic{}, keeper: k}
}

// Name returns the x/tfhe module's name.
func (AppModule) Name() string { return types.ModuleName }

// RegisterServices registers the gRPC Msg + Query servers with the
// configured router. The MsgServer routes EncryptedDeposit /
// ComputeOnEncrypted etc. through the keeper; Query server exposes
// CiphertextById + Params (others stubbed pending Stage 5.3).
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQuerier(am.keeper))
}

// InitGenesis performs genesis initialization for the module.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) {
	var gs types.GenesisState
	cdc.MustUnmarshalJSON(data, &gs)
	InitGenesis(ctx, am.keeper, &gs)
}

// ExportGenesis returns the exported genesis state as raw bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(gs)
}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// _ ensures the unused-context import warning is silenced if all
// places using context get refactored away. context.Context is part
// of every gRPC method on this module, so this is a defensive guard.
var _ = context.Background
