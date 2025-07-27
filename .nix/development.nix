# Development environment for devcmd project
# Dogfooding our own tool for development commands
{ pkgs, self ? null, gitRev ? "dev", system }:
let
  # Import our own library to create the development CLI
  devcmdLib = import ./lib.nix {
    inherit pkgs self gitRev system;
    lib = pkgs.lib;
  };
  
  # Generate the development CLI from our commands.cli file
  devCLI =
    if self != null then
      devcmdLib.mkDevCLI
        {
          name = "dev";
          binaryName = "dev"; # Explicitly set binary name for self-awareness
          commandsFile = ../commands.cli;
          version = "latest";
          meta = {
            description = "Devcmd development CLI - dogfooding our own tool";
            longDescription = ''
              This CLI is generated from commands.cli using devcmd itself.
              It provides a streamlined development experience with all
              necessary commands for building, testing, and maintaining devcmd.
            '';
          };
        }
    else
      null;
in
pkgs.mkShell {
  name = "devcmd-dev";
  buildInputs = with pkgs; [
    # Core Go development
    go
    gopls
    golangci-lint
    # Development tools
    git
    zsh
    # Code formatting
    nixpkgs-fmt
    gofumpt
  ] ++ pkgs.lib.optional (devCLI != null) devCLI;
  shellHook = ''
    echo "🔧 Devcmd Development Environment"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "🔨 Build devcmd first:"
    echo "  cd cli && go build -o ../devcmd ./main.go"
    echo ""
    echo "🚀 Then use the CLI:"
    echo "  ./devcmd run build    # Build the project"
    echo "  ./devcmd run test     # Run all tests"
    echo "  ./devcmd run help     # See all commands"
    echo ""
    echo "💡 Or use direct Go commands:"
    echo "  go test ./core/... ./runtime/... ./cli/... # Test all modules"
    exec ${pkgs.zsh}/bin/zsh
  '';
}
