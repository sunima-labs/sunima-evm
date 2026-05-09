package types

// Msg names for routing — keep stable, used in tx route.
const (
	TypeMsgEncryptedDeposit       = "encrypted_deposit"
	TypeMsgComputeOnEncrypted     = "compute_on_encrypted"
	TypeMsgRequestDecryption      = "request_decryption"
	TypeMsgSubmitAttestation      = "submit_attestation"
	TypeMsgRegisterAttester       = "register_attester"
)

// MsgEncryptedDeposit submits a ciphertext to the module's vault.
// Replaces proto-generated struct until buf codegen is wired up.
type MsgEncryptedDeposit struct {
	Sender     string // bech32 address
	Ciphertext []byte // serialized tfhe-rs ciphertext
}

// MsgComputeOnEncrypted requests a homomorphic operation on stored ciphertexts.
// OperationType: "add" | "compare" | "conditional"
type MsgComputeOnEncrypted struct {
	Sender         string
	InputIDs       [][]byte // ciphertext content hashes
	OperationType  string
	OutputOwner    string // bech32 address that will own the result
}

// MsgRequestDecryption requests threshold decryption of a ciphertext.
// Decryption proceeds once a quorum of attesters submits valid partials.
type MsgRequestDecryption struct {
	Sender       string
	CiphertextID []byte
	Recipient    string // bech32 address that receives plaintext result
}

// MsgSubmitAttestation contributes one partial decryption + signature toward a quorum.
type MsgSubmitAttestation struct {
	Attester       string // bech32 address of registered attester
	RequestID      []byte // matches a pending DecryptionRequest
	PartialDecrypt []byte // share's contribution
	Signature      []byte // ed25519 over (RequestID || PartialDecrypt)
}

// MsgRegisterAttester adds a new attester to the quorum (governance-gated).
type MsgRegisterAttester struct {
	Authority    string // governance module address
	AttesterAddr string // bech32 address of the new attester
	PubKey       []byte // ed25519 pubkey for attestation signatures
	SharePubkey  []byte // tfhe-rs threshold share verification key
}
