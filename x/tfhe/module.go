package tfhe

import (
	"github.com/sunima-labs/sunima-evm/x/tfhe/keeper"
	"github.com/sunima-labs/sunima-evm/x/tfhe/types"
)

// AppModule implements the Cosmos SDK module.AppModule interface.
//
// Method signatures match Cosmos SDK v0.50+ but are commented out until
// SDK is added to go.mod. Real wiring happens after `nix develop` provides protoc/buf
// and the proto layer compiles.
type AppModule struct {
	keeper keeper.Keeper
}

// NewAppModule constructs the module from a Keeper.
func NewAppModule(k keeper.Keeper) AppModule {
	return AppModule{keeper: k}
}

// Name returns the module's name as registered with the SDK.
func (AppModule) Name() string { return types.ModuleName }

// RegisterServices is the integration point for Msg/Query servers.
// Real impl will register sunima.tfhe.v1.MsgServer + QueryServer here.
func (am AppModule) RegisterServices( /* cfg module.Configurator */ ) {
	// TODO: types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	// TODO: types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQuerier(am.keeper))
}
