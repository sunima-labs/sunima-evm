package keeper

import (
	"github.com/sunima-labs/sunima-evm/x/tfhe/types"
)

// msgServer implements the (planned) proto MsgServer interface.
// Once buf codegen lands, this satisfies sunima.tfhe.v1.MsgServer.
type msgServer struct {
	keeper Keeper
}

// NewMsgServerImpl wraps a Keeper into the MsgServer.
func NewMsgServerImpl(keeper Keeper) msgServer {
	return msgServer{keeper: keeper}
}

// EncryptedDeposit handler. Returns ciphertext content hash on success.
func (s msgServer) EncryptedDeposit(msg types.MsgEncryptedDeposit) ([]byte, error) {
	return s.keeper.StoreCiphertext(msg.Ciphertext, msg.Sender)
}

// ComputeOnEncrypted handler.
func (s msgServer) ComputeOnEncrypted(msg types.MsgComputeOnEncrypted) ([]byte, error) {
	return s.keeper.HomomorphicCompute(msg.OperationType, msg.InputIDs)
}

// RequestDecryption handler. Records a pending DecryptionRequest awaiting quorum.
func (s msgServer) RequestDecryption(msg types.MsgRequestDecryption) error {
	// TODO: persist request, emit event so attesters pick it up
	_ = msg
	return nil
}

// SubmitAttestation handler. Accumulates partials toward a 5-of-9 quorum.
func (s msgServer) SubmitAttestation(msg types.MsgSubmitAttestation) error {
	// TODO: append partial to request, on threshold reached → combine + deliver plaintext
	_ = msg
	return nil
}

// RegisterAttester handler. Governance-gated.
func (s msgServer) RegisterAttester(msg types.MsgRegisterAttester) error {
	// TODO: authority check, persist attester, cap total at 9
	_ = msg
	return nil
}
