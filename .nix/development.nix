# Development environment for devcmd project - smart derivation approach
{ pkgs, self ? null, gitRev ? "dev", system }:
let
  # Import our library to create the development CLI using fixed-output derivation
  devcmdLib = import ./lib.nix {
    inherit pkgs self gitRev system;
    lib = pkgs.lib;
  };

  # Build the dev CLI using fixed-output derivation (allows network access)
  devCLI = devcmdLib.mkDevCLI {
    name = "devcmd-dev-cli";
    binaryName = "dev";
    commandsFile = ../commands.cli;
    version = "dev-${gitRev}";
  };
in
pkgs.mkShell {
  name = "devcmd-dev";
  
  buildInputs = with pkgs; [
    # Development tools
    go
    gopls
    golangci-lint
    git
    zsh
    nixpkgs-fmt
    gofumpt
  ] ++ [ 
    self.packages.${system}.devcmd  # Include the devcmd binary itself
    devCLI                          # Include the generated dev CLI (built via fixed-output derivation)
  ];
  
  shellHook = ''
    echo "🔧 Devcmd Development Environment"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "Available commands:"
    echo "  devcmd - The devcmd CLI generator"
    echo "  dev    - Development commands for this project (built via fixed-output derivation)"
    echo ""
    echo "The dev CLI is automatically built using a fixed-output derivation"
    echo "that allows network access for Go module downloads while maintaining"
    echo "reproducibility through content hashing."
    echo ""
    echo "Run 'dev help' to see available development commands"
    exec ${pkgs.zsh}/bin/zsh
  '';
}
