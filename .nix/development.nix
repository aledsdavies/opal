# Development environment for devcmd project
# Dogfooding our own tool for development commands
{ pkgs, self ? null, gitRev ? "dev" }:
let
  # For now, skip the generated CLI and use manual commands
  # TODO: Re-enable when Nix package build works with Go modules
  devCLI = null;
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
