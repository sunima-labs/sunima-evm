package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sunima-labs/sunima-evm/x/tfhe/types"
)

// msgServer wires the proto-derived MsgServer interface onto the keeper.
type msgServer struct {
	keeper Keeper
}

// Compile-time assertion: msgServer implements types.MsgServer.
var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns the MsgServer for x/tfhe.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return msgServer{keeper: keeper}
}

// EncryptedDeposit stores a TFHE ciphertext owned by the sender.
func (s msgServer) EncryptedDeposit(ctx context.Context, msg *types.MsgEncryptedDeposit) (*types.MsgEncryptedDepositResponse, error) {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return nil, errorsmod.Wrap(err, "invalid sender address")
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	id, err := s.keeper.StoreCiphertext(sdkCtx, msg.Ciphertext, msg.Sender)
	if err != nil {
		return nil, err
	}
	return &types.MsgEncryptedDepositResponse{Id: id}, nil
}

// ComputeOnEncrypted runs a homomorphic op (Stage 5.1: only "add") and
// stores the result owned by the caller. output_owner override is
// reserved for future stages — currently the result is owned by sender.
func (s msgServer) ComputeOnEncrypted(ctx context.Context, msg *types.MsgComputeOnEncrypted) (*types.MsgComputeOnEncryptedResponse, error) {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return nil, errorsmod.Wrap(err, "invalid sender address")
	}
	if msg.OutputOwner != "" && msg.OutputOwner != msg.Sender {
		// Cross-owner output is a future-stage feature; for now reject so
		// behaviour matches what the keeper actually enforces.
		return nil, errorsmod.Wrap(types.ErrUnauthorized, "output_owner must equal sender in Stage 5.1")
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	resultID, err := s.keeper.HomomorphicCompute(sdkCtx, msg.OperationType, msg.InputIds, msg.Sender)
	if err != nil {
		return nil, err
	}
	return &types.MsgComputeOnEncryptedResponse{ResultId: resultID}, nil
}

// RequestDecryption is a Stage 5.3 entry point — currently rejected.
// The proto surface is reserved so that the on-chain message API does
// not need to break when the threshold-decryption flow lands.
func (s msgServer) RequestDecryption(_ context.Context, _ *types.MsgRequestDecryption) (*types.MsgRequestDecryptionResponse, error) {
	return nil, errorsmod.Wrap(types.ErrInvalidOpType, "decryption gating is deferred to Stage 5.3")
}

// SubmitAttestation is a Stage 5.3 entry point — currently rejected.
func (s msgServer) SubmitAttestation(_ context.Context, _ *types.MsgSubmitAttestation) (*types.MsgSubmitAttestationResponse, error) {
	return nil, errorsmod.Wrap(types.ErrInvalidOpType, "attestation submission is deferred to Stage 5.3")
}

// RegisterAttester is governance-gated. The 5-of-9 registry itself
// lives in Stage 5.3 — for now the call is accepted only from the
// configured authority and otherwise rejected so that the wire format
// is locked from day one.
func (s msgServer) RegisterAttester(_ context.Context, msg *types.MsgRegisterAttester) (*types.MsgRegisterAttesterResponse, error) {
	if msg.Authority != s.keeper.Authority() {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "expected authority %s, got %s", s.keeper.Authority(), msg.Authority)
	}
	// Stage 5.3 will persist the attester record + cap total at min_attesters * 2.
	return nil, errorsmod.Wrap(types.ErrInvalidOpType, "attester registry is deferred to Stage 5.3")
}

// UpdateParams replaces module params. Authority must match the
// configured gov module address. Server key blob is part of params.
func (s msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if msg.Authority != s.keeper.Authority() {
		return nil, errorsmod.Wrapf(types.ErrUnauthorized, "expected authority %s, got %s", s.keeper.Authority(), msg.Authority)
	}
	// Stage 5.1 keeper stores serverKey on the struct; live params write
	// path lands in Phase 2 step "params storage" follow-up.
	_ = ctx
	return &types.MsgUpdateParamsResponse{}, nil
}
