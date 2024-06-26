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
          k9s
          rclone
          kind
          yq
          kubernetes-helm
          helmfile
          go
          kpkgs.helm-readme-generator
        ];
      };
    }
  );
}
