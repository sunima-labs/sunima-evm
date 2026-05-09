package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary x/tfhe interfaces and
// concrete types on the provided LegacyAmino codec. Stage 5.1 ships
// without amino bindings — most modern Cosmos SDK chains rely on
// proto/cdc-only flow, and the existing tx wrappers do not need amino
// serialisation. We register the Msg types so that legacy clients that
// still introspect via amino can at least see them.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgEncryptedDeposit{}, "sunima/x/tfhe/MsgEncryptedDeposit", nil)
	cdc.RegisterConcrete(&MsgComputeOnEncrypted{}, "sunima/x/tfhe/MsgComputeOnEncrypted", nil)
	cdc.RegisterConcrete(&MsgRequestDecryption{}, "sunima/x/tfhe/MsgRequestDecryption", nil)
	cdc.RegisterConcrete(&MsgSubmitAttestation{}, "sunima/x/tfhe/MsgSubmitAttestation", nil)
	cdc.RegisterConcrete(&MsgRegisterAttester{}, "sunima/x/tfhe/MsgRegisterAttester", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "sunima/x/tfhe/MsgUpdateParams", nil)
}

// RegisterInterfaces registers the x/tfhe interfaces and concrete
// implementations with the provided codec interface registry.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgEncryptedDeposit{},
		&MsgComputeOnEncrypted{},
		&MsgRequestDecryption{},
		&MsgSubmitAttestation{},
		&MsgRegisterAttester{},
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
