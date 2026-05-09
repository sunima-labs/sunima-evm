{
  description = "Sunima Cosmos EVM chain — fork of cosmos/evm with TFHE module";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";

    # Pinned cosmos-evm fork — local clone for dev (SSH path for CI later)
    # Local clone must exist at /root/projects/cosmos-evm at the expected rev.
    # Sync via scripts/sync-cosmos-evm.sh.
    cosmos-evm = {
      url = "git+file:///root/projects/cosmos-evm?ref=main&rev=e118daef885f4a43f456cf881fa9fe6806778c6d";
      flake = false;
    };
  };

  outputs = { self, nixpkgs, flake-utils, cosmos-evm }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        goVersion = pkgs.go_1_25 or pkgs.go;
      in {
        devShells.default = pkgs.mkShell {
          buildInputs = [
            goVersion
            pkgs.gnumake
            pkgs.git
            pkgs.protobuf
            pkgs.buf
            pkgs.golangci-lint
            pkgs.jq
            # Rust toolchain for tfhe-rs FFI bridge
            pkgs.rustc
            pkgs.cargo
            pkgs.clang
            pkgs.pkg-config
          ];
          shellHook = ''
            export COSMOS_EVM_SRC=${cosmos-evm}
            export CGO_ENABLED=1
            # protoc-gen-gocosmos and protoc-gen-grpc-gateway are go-installed
            # under $GOPATH/bin (default /root/go/bin); buf needs them on PATH
            # to run `buf generate` against the proto/ tree.
            export PATH="$(go env GOPATH)/bin:$PATH"
            echo "Sunima EVM dev shell"
            echo "  go:    $(go version)"
            echo "  rustc: $(rustc --version)"
            echo "  cargo: $(cargo --version)"
            echo "  cosmos-evm pinned: $COSMOS_EVM_SRC"
          '';
        };

        packages.cosmos-evm-src = pkgs.stdenv.mkDerivation {
          name = "cosmos-evm-source-e118daef885f4a43f456cf881fa9fe6806778c6d";
          src = cosmos-evm;
          dontBuild = true;
          installPhase = "cp -r . $out";
        };
      }
    );
}
