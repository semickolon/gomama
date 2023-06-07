{
  description = "gomama";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-23.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let 
        pkgs = import nixpkgs { inherit system; };
        gomama = pkgs.buildGoModule rec {
          name = "gomama";
          src = ./.;
          vendorHash = "sha256-u+8StYUrSA6chPtbNzqYB5/o07PIYfZOzsw5Q7rqNho=";
        };
      in
      {
        packages.default = gomama;
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            go-outline
            delve
            go-tools
          ];
        };
      }
    );
}