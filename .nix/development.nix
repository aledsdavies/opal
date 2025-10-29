# Development environment for Opal project - interpreter mode only
{ pkgs, self ? null, gitRev ? "dev", system }:

pkgs.mkShell {
  name = "opal-dev";

  buildInputs = with pkgs; [
    # Development tools
    go
    gopls
    golangci-lint
    git
    zsh
    nixpkgs-fmt
    gofumpt
    openssh  # For SSH session testing
  ] ++ (if self != null then [
    self.packages.${system}.opal # Include the opal binary itself
  ] else []);

  shellHook = ''
    # Only show welcome message in interactive shells
    if [ -t 0 ]; then
      echo "🔧 Opal Development Environment"
      echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      echo ""
      echo "Available tools:"
      echo "  opal       - The Opal CLI (operations planning language)"
      echo "  go         - Go compiler and tools"
      echo "  gofumpt    - Go formatter"
      echo "  golangci-lint - Go linter"
      echo "  nixpkgs-fmt   - Nix formatter"
      echo ""
      echo "Development commands (run manually):"
      echo "  go fmt ./...                    - Format Go code"
      echo "  gofumpt -w .                   - Format with gofumpt"
      echo "  golangci-lint run              - Run linter"
      echo "  go test -v ./...               - Run all tests"
      echo "  go test -v -short ./...        - Run tests (skip SSH integration)"
      echo "  cd cli && go build -o opal .   - Build CLI"
      echo ""
      echo "SSH Testing (requires SSH server on localhost):"
      echo "  SSH tests skip gracefully if localhost SSH unavailable"
      echo "  To enable: ensure SSH server running and key-based auth configured"
      echo ""
      echo "Opal usage:"
      echo "  opal deploy --dry-run          - Show execution plan"
      echo "  opal deploy                    - Execute operation"
      echo ""
    fi
  '';
}
