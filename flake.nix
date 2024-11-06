{
  inputs = {
    utils.url = "github:numtide/flake-utils";
    nixpkgs.url = "github:nixos/nixpkgs";
    knixpkgs.url = "github:karitham/knixpkgs";
  };
  outputs = {
    self,
    nixpkgs,
    knixpkgs,
    utils,
  }:
    utils.lib.eachDefaultSystem (
      system: let
        pkgs = nixpkgs.legacyPackages.${system};
        kpkgs = knixpkgs.packages.${system};
      in {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            kubectl
            rclone
            kind
            kubernetes-helm
            helmfile
            go
            gofumpt
          ];
        };

        packages = {
          readme = pkgs.writeShellScriptBin "readme" "${kpkgs.helm-readme-generator}/bin/readme-generator --readme README.md --values chart/values.yaml";
        };
      }
    );
}
