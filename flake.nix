{
  description = "Sunima Cosmos EVM chain — fork of cosmos/evm with TFHE module";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";

    # Pinned cosmos-evm fork — sync via update-cosmos-evm.sh
    cosmos-evm = {
      url = "git+ssh://git@github.com/sunima-labs/cosmos-evm.git?ref=main&rev=e118daef885f4a43f456cf881fa9fe6806778c6d";
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
          ];
          shellHook = ''
            export COSMOS_EVM_SRC=${cosmos-evm}
            echo "Sunima EVM dev shell"
            echo "  go: $(go version)"
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
