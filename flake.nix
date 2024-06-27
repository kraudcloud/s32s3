{
  inputs = {
    utils.url = "github:numtide/flake-utils";
    nixpkgs.url = "github:nixos/nixpkgs";
    knixpkgs.url = "github:karitham/knixpkgs";
  };
  outputs = { self, nixpkgs, knixpkgs, utils }: utils.lib.eachDefaultSystem (system:
    let
      pkgs = nixpkgs.legacyPackages.${system};
      kpkgs = knixpkgs.packages.${system};
    in
    {
      devShell = pkgs.mkShell {
        buildInputs = with pkgs; [
          kubectl
          rclone
          kind
          kubernetes-helm
          helmfile
          go
          gofumpt
          kpkgs.helm-readme-generator
        ];
      };
    }
  );
}
