package types

const (
	// ModuleName defines the module name.
	ModuleName = "tfhe"

	// StoreKey is the default store key for the module.
	StoreKey = ModuleName

	// RouterKey is the message route for the module.
	RouterKey = ModuleName

	// QuerierRoute is the querier route for the module.
	QuerierRoute = ModuleName
)

// Storage key prefixes — see x/tfhe/README.md "State layout".
var (
	CiphertextKeyPrefix         = []byte{0x01}
	OwnershipIndexKeyPrefix     = []byte{0x02}
	AttesterKeyPrefix           = []byte{0x03}
	DecryptionRequestKeyPrefix  = []byte{0x04}
	ParamsKey                   = []byte{0x05}
)
