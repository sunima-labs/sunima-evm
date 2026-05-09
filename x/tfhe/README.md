# x/tfhe — Sunima TFHE AppModule

Cosmos SDK module that exposes Threshold FHE primitives as on-chain operations.

## Scope

- Encrypted deposits (ciphertext into module-managed vault)
- Homomorphic compute over ciphertexts (add / compare / conditional)
- Threshold decryption gated by an attestation quorum

## Architecture

```
EVM (cosmos-evm)        x/tfhe Cosmos module        tfhe-rs worker pool
─────────────────       ────────────────────        ──────────────────
contract calls   ───►   precompile dispatch ───►   FFI Go ↔ Rust
                                ▼                          │
                          state writes                     │
                                ▼                  ◄──── attestation
                          attestation gate (5-of-9)
                                ▼
                          msg server / events
```

## 5-of-9 attestation pattern (planned)

Decryption requires 5 of 9 registered attesters to sign a `DecryptionAuthorization`. Each attester runs a `tfhe-rs` worker with a share of the threshold key. The module verifies the aggregated signature on `Msg.DecryptWithAttestation`.

## tfhe-rs FFI bridge (planned)

A separate `internal/tfhebridge/` package wraps `tfhe-rs` (Rust) via cgo. The bridge exposes:

- `Encrypt(plaintext, pubkey) ciphertext`
- `Add(a, b) ciphertext`
- `Compare(a, b) ciphertext` (encrypted boolean)
- `PartialDecrypt(ct, share) partial`
- `CombinePartials(partials []partial) plaintext`

Runs in a worker pool, called from the module via channels, never inline in consensus.

## State layout

| Key prefix | Value | Notes |
|-----------|-------|-------|
| `0x01 \| ciphertext_id` | `Ciphertext` | indexed by content hash |
| `0x02 \| owner_addr \| ciphertext_id` | `bool` | ownership index |
| `0x03 \| attester_id` | `Attester` | registered quorum members |
| `0x04 \| request_id` | `DecryptionRequest` | pending decryptions awaiting quorum |
| `0x05` | `Params` | module parameters |

## Files

- `module.go` — `AppModule` and `AppModuleBasic` implementations
- `genesis.go` — InitGenesis / ExportGenesis
- `keeper/keeper.go` — state access, business logic
- `keeper/msg_server.go` — Msg handlers
- `types/` — Msg structs, keys, errors, params
- `proto/sunima/tfhe/v1/` — proto definitions (codegen via `buf generate`, runs under `nix develop`)

## Status

Scaffold only. No proto codegen yet, no FFI bridge, no actual ciphertext handling. Follow-up tasks tracked in the working memory.
