# Development environment for devcmd project - smart derivation approach
{ pkgs, self ? null, gitRev ? "dev", system }:
let
  # Import our library to create the development CLI using fixed-output derivation
  devcmdLib = import ./lib.nix {
    inherit pkgs self gitRev system;
    lib = pkgs.lib;
  };

  # TODO: Re-enable dev CLI generation once FOD store path issue is resolved
  # For now, users can manually run: devcmd build --file commands.cli --binary dev
  # devCLI = devcmdLib.mkDevCLI {
  #   name = "devcmd-dev-cli";
  #   binaryName = "dev";
  #   commandsFile = ../commands.cli;
  #   version = "dev-${gitRev}";
  # };
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
    # devCLI                          # TODO: Re-enable when FOD issue is resolved
  ];
  
  shellHook = ''
    echo "🔧 Devcmd Development Environment"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "Available commands:"
    echo "  devcmd - The devcmd CLI generator"
    echo ""
    echo "To generate the dev CLI, run:"
    echo "  devcmd build --file commands.cli --binary dev -o dev"
    echo ""
    echo "Then you can run './dev help' to see development commands"
    echo ""
    echo "Note: Auto-generated dev CLI is temporarily disabled due to"
    echo "      FOD store path reference issues. Will be re-enabled with"
    echo "      dynamic derivations when they become stable."
    exec ${pkgs.zsh}/bin/zsh
  '';
}
