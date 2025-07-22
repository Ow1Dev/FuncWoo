{
  description = "Configuration for NoctiFunc";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs = inputs @ {flake-parts, ...}:
    flake-parts.lib.mkFlake {inherit inputs;} {
      systems = ["x86_64-linux" "aarch64-linux" "aarch64-darwin" "x86_64-darwin"];

      perSystem = {
        config,
        self',
        inputs',
        pkgs,
        system,
        ...
      }: let
        name = "NoctiFunc";
        vendorHash = "sha256-int+VOY11p/Vg/ZVKf2I37AejHvH0EUZJoW+U8EFVbQ=";
      in {
        devShells = {
          default = pkgs.mkShell {
            inputsFrom = [self'.packages.default];
            buildInputs = with pkgs; [
              go_1_24
              golangci-lint
              protobuf
              protoc-gen-go
              protoc-gen-go-grpc
              grpcurl
            ];
          };
        };

        packages = {
          default = pkgs.buildGoModule {
            inherit name vendorHash;
            src = ./.;
            subPackages = ["cmd"];
            postBuild = ''
              mv $GOPATH/bin/cmd $GOPATH/bin/noctifunc
            '';
          };
        };
      };
    };
}
