# sunima-evm

Sunima sovereign Cosmos EVM chain — built on top of a pinned cosmos/evm fork.

## Layout

This repo is the Sunima-specific working tree:

- `flake.nix` — Nix flake with pinned cosmos-evm input (deterministic builds)
- `scripts/sync-cosmos-evm.sh` — utility to fast-forward the cosmos-evm mirror against upstream cosmos/evm
- `x/tfhe/` (planned) — Sunima TFHE precompile / AppModule

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
