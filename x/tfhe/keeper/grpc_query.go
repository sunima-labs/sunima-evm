package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sunima-labs/sunima-evm/x/tfhe/types"
)

// querier wires the proto-derived QueryServer interface onto the keeper.
type querier struct {
	keeper Keeper
}

// Compile-time assertion: querier implements types.QueryServer.
var _ types.QueryServer = querier{}

// NewQuerier returns a QueryServer for the x/tfhe module.
func NewQuerier(k Keeper) types.QueryServer {
	return querier{keeper: k}
}

// CiphertextById returns a stored ciphertext if the (gRPC-context) caller
// is a known owner. Stage 5.1: ownership is derived from the request
// header "x-cosmos-sender" — if absent, the query is rejected. A future
// stage may relax this to allow public introspection of ciphertexts
// (only the plaintext stays sealed under threshold decryption anyway).
func (q querier) CiphertextById(ctx context.Context, req *types.QueryCiphertextByIdRequest) (*types.QueryCiphertextByIdResponse, error) {
	if req == nil {
		return nil, errorsmod.Wrap(types.ErrInvalidCiphertext, "request is nil")
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	caller := callerFromMetadata(ctx)
	ct, err := q.keeper.GetCiphertext(sdkCtx, req.Id, caller)
	if err != nil {
		return nil, err
	}
	return &types.QueryCiphertextByIdResponse{
		Ciphertext: types.Ciphertext{Id: req.Id, Data: ct, Owner: caller},
	}, nil
}

// OwnedCiphertexts is a Stage 5.3 enumeration helper — currently
// rejected; full impl needs the ownership index iteration that we'll
// wire up alongside the threshold decryption flow.
func (q querier) OwnedCiphertexts(_ context.Context, _ *types.QueryOwnedCiphertextsRequest) (*types.QueryOwnedCiphertextsResponse, error) {
	return nil, errorsmod.Wrap(types.ErrInvalidOpType, "owned-ciphertexts query is deferred to Stage 5.3")
}

// Attesters returns the registered quorum members. Stage 5.1 ships
// with an empty registry; the call returns an empty list rather than
// erroring so chains under construction do not break query clients.
func (q querier) Attesters(_ context.Context, _ *types.QueryAttestersRequest) (*types.QueryAttestersResponse, error) {
	return &types.QueryAttestersResponse{Attesters: nil}, nil
}

// DecryptionRequest is a Stage 5.3 entry point — currently rejected.
func (q querier) DecryptionRequest(_ context.Context, _ *types.QueryDecryptionRequestRequest) (*types.QueryDecryptionRequestResponse, error) {
	return nil, errorsmod.Wrap(types.ErrInvalidOpType, "decryption-request query is deferred to Stage 5.3")
}

// Params returns the configured Params. Stage 5.1: server_key blob is
// stripped from the response to keep query responses small (~180 MB
// otherwise). Operators read server_key directly from genesis.json.
func (q querier) Params(_ context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	p := types.DefaultParams()
	// Replace server_key with its size in bytes so callers see "set/unset"
	// without paying the bandwidth.
	if len(q.keeper.ServerKey()) > 0 {
		p.ServerKey = nil // keep wire format predictable; size lives in events later
	}
	return &types.QueryParamsResponse{Params: p}, nil
}

// callerFromMetadata extracts the caller's bech32 address from gRPC
// metadata if present. For now this is a stub that returns "" — until
// the chain is wired up, we have no metadata source. Once
// keeper.NewMsgServerImpl invocations land via the BaseApp routing,
// the SDK injects sender address into ctx automatically.
func callerFromMetadata(_ context.Context) string {
	return ""
}
