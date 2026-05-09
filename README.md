# sunima-evm

Sunima sovereign Cosmos EVM chain — built on top of a pinned cosmos/evm fork.

## Layout

- `flake.nix` — Nix flake with pinned cosmos-evm input (deterministic builds)
- `scripts/sync-cosmos-evm.sh` — utility to fast-forward the cosmos-evm mirror against upstream cosmos/evm
- `x/tfhe/` — Sunima TFHE AppModule (scaffold; see `x/tfhe/README.md` for design)

The cosmos-evm mirror lives in a separate private repo at `sunima-labs/cosmos-evm`. That repo is updated by syncing from `upstream` (cosmos/evm). This repo pins to a specific commit hash for reproducibility.

## Dev shell

```
nix develop
```

Provides Go, protoc, buf, golangci-lint.

## Sync upstream

```
./scripts/sync-cosmos-evm.sh           # dry-run
./scripts/sync-cosmos-evm.sh --apply   # fast-forward and push
```

Then bump the `rev` field in `flake.nix` to the new commit hash.

## Modules

- [`x/tfhe`](x/tfhe/README.md) — Threshold FHE primitives (encrypted deposits, homomorphic compute, 5-of-9 attestation-gated decryption). Scaffold only; proto codegen and tfhe-rs FFI bridge land in follow-up.

## Status

Scaffold stage. Module skeletons compile-ready as soon as Cosmos SDK + cosmos-evm dependencies are added to `go.mod`. No proto codegen yet — proto files in `x/*/proto/` will be generated via `buf generate` once the dev shell is exercised.
