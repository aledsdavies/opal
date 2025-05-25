# Devcmd - Development Command DSL Parser
# Run `just` to see all available commands
# Assumes you're already in `nix develop` shell

# Variables
project_name := "devcmd"
grammar_dir := "grammar"
gen_dir := "internal/gen"
parser_pkg := "github.com/aledsdavies/devcmd/pkgs/parser"
generator_pkg := "github.com/aledsdavies/devcmd/pkgs/generator"
examples_dir := "examples"

# Default command - show available commands with descriptions
default:
    @echo "🔧 Devcmd Development Commands"
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @echo ""
    @echo "🚀 Quick Start:"
    @echo "  setup          - Initial project setup (grammar generation)"
    @echo "  build          - Build the CLI tool"
    @echo "  test           - Run all tests"
    @echo "  lint           - Run linters and code quality checks"
    @echo ""
    @echo "📝 Grammar & Parsing:"
    @echo "  grammar        - Generate parser from ANTLR grammar (if needed)"
    @echo "  parse FILE     - Parse a devcmd file and show AST"
    @echo "  validate FILE  - Validate a devcmd file"
    @echo ""
    @echo "🔨 Code Generation:"
    @echo "  generate FILE  - Generate Go CLI from devcmd file"
    @echo "  compile FILE   - Parse, generate, and compile Go CLI"
    @echo ""
    @echo "🧪 Testing & Quality:"
    @echo "  test-parser    - Run parser tests only"
    @echo "  test-generator - Run generator tests only"
    @echo "  test-all       - Run comprehensive test suite"
    @echo "  test-coverage  - Run tests with coverage"
    @echo "  benchmark      - Run performance benchmarks"
    @echo ""
    @echo "📦 Nix Integration:"
    @echo "  nix-build      - Build all Nix packages"
    @echo "  nix-examples   - Build all example CLIs with Nix"
    @echo "  nix-test       - Run Nix-based tests"
    @echo "  nix-check      - Run nix flake check"
    @echo "  try-examples   - Try all example CLIs interactively"
    @echo ""
    @echo "🧹 Maintenance:"
    @echo "  clean          - Clean generated files and build artifacts"
    @echo "  format         - Format all code"
    @echo ""
    @echo "For detailed help: just --list"
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# =============================================================================
# 🚀 QUICK START COMMANDS
# =============================================================================

# Initial project setup
setup:
    @echo "🔧 Setting up Devcmd development environment..."
    just grammar
    go mod tidy
    go mod download
    @echo "✅ Setup complete! Run 'just test' to verify everything works."

# Build the CLI tool
build:
    @echo "🔨 Building devcmd CLI..."
    go build -o {{project_name}} ./cmd/{{project_name}}

# Run all tests
test:
    @echo "🧪 Running all tests..."
    go test -v ./...

# =============================================================================
# 📝 GRAMMAR & PARSING COMMANDS
# =============================================================================

# Generate parser from ANTLR grammar (only if files don't exist or are outdated)
grammar:
    @echo "📝 Checking ANTLR grammar..."
    @if [ ! -f {{gen_dir}}/devcmd_lexer.go ] || [ {{grammar_dir}}/devcmd.g4 -nt {{gen_dir}}/devcmd_lexer.go ]; then \
        echo "Generating parser from ANTLR grammar..."; \
        mkdir -p {{gen_dir}}; \
        cd {{grammar_dir}} && antlr -Dlanguage=Go -package gen -o ../{{gen_dir}} devcmd.g4; \
        echo "✅ Parser generated successfully"; \
    else \
        echo "✅ Generated parser files are up to date"; \
    fi

# Force regenerate grammar (for development)
grammar-force:
    @echo "📝 Force regenerating parser from ANTLR grammar..."
    mkdir -p {{gen_dir}}
    cd {{grammar_dir}} && antlr -Dlanguage=Go -package gen -o ../{{gen_dir}} devcmd.g4
    @echo "✅ Parser regenerated successfully"

# Parse a devcmd file and show AST
parse FILE:
    @echo "🔍 Parsing {{FILE}}..."
    go run ./cmd/{{project_name}} parse {{FILE}}

# Validate a devcmd file
validate FILE:
    @echo "✅ Validating {{FILE}}..."
    go run ./cmd/{{project_name}} validate {{FILE}}

# =============================================================================
# 🔨 CODE GENERATION COMMANDS
# =============================================================================

# Generate Go CLI from devcmd file
generate FILE:
    @echo "🔨 Generating Go CLI from {{FILE}}..."
    go run ./cmd/{{project_name}} generate {{FILE}}

# Parse, generate, and compile Go CLI in one step
compile FILE:
    @echo "⚡ Compiling {{FILE}} to executable..."
    go run ./cmd/{{project_name}} compile {{FILE}}

# =============================================================================
# 🧪 TESTING & QUALITY COMMANDS
# =============================================================================

# Run parser tests only
test-parser:
    @echo "🧪 Running parser tests..."
    go test -v {{parser_pkg}}

# Run generator tests only
test-generator:
    @echo "🧪 Running generator tests..."
    go test -v {{generator_pkg}}

# Run comprehensive test suite
test-all: test-parser test-generator test-coverage

# Run tests with coverage
test-coverage:
    @echo "🧪 Running tests with coverage..."
    go test -race -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Run performance benchmarks
benchmark:
    @echo "⚡ Running performance benchmarks..."
    go test -bench=. -benchmem ./...

# Run linters and code quality checks
lint:
    @echo "🔍 Running linters..."
    golangci-lint run
    @echo "🔍 Checking grammar for issues..."
    antlr -Xlog {{grammar_dir}}/devcmd.g4 || echo "Grammar check complete"

# =============================================================================
# 📦 NIX INTEGRATION COMMANDS
# =============================================================================

# Build all Nix packages
nix-build:
    @echo "📦 Building all Nix packages..."
    nix build .#devcmd
    nix build .#basicDev
    @echo "✅ All packages built"

# Build example CLIs with Nix
nix-examples:
    @echo "🎯 Building example CLIs with Nix..."
    nix build .#basicDev .#webDev .#goProject .#rustProject .#dataScienceProject .#devOpsProject
    @echo "✅ Example CLIs built"

# Run Nix-based tests
nix-test:
    @echo "🧪 Running Nix tests..."
    nix build .#tests
    nix build .#test-examples
    @echo "✅ All Nix tests passed"

# Run nix flake check
nix-check:
    @echo "🔍 Running comprehensive Nix checks..."
    nix flake check --show-trace
    @echo "✅ All checks passed"

# Update flake lock file
nix-update:
    @echo "🔄 Updating flake inputs..."
    nix flake update
    @echo "✅ Flake updated"

# Try all example CLIs interactively
try-examples:
    @echo "🎯 Interactive Example CLI Testing"
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @echo ""
    @echo "1. Basic Development CLI:"
    @echo "   nix run .#basicDev -- --help"
    @nix run .#basicDev -- --help
    @echo ""
    @echo "2. Web Development CLI:"
    @echo "   nix run .#webDev -- --help"
    @nix run .#webDev -- --help
    @echo ""
    @echo "3. Go Project CLI:"
    @echo "   nix run .#goProject -- --help"
    @nix run .#goProject -- --help
    @echo ""
    @echo "🎉 Try running: nix run .#basicDev -- build"

# Show available Nix outputs
nix-show:
    @echo "📋 Available Nix flake outputs:"
    nix flake show

# Enter specific development shells
shell-basic:
    @echo "🐚 Entering basic development shell..."
    nix develop .#basicShell

shell-web:
    @echo "🌐 Entering web development shell..."
    nix develop .#webShell

shell-go:
    @echo "🐹 Entering Go development shell..."
    nix develop .#goShell

shell-data:
    @echo "📊 Entering data science shell..."
    nix develop .#dataShell

shell-test:
    @echo "🧪 Entering test environment..."
    nix develop .#testEnv

# =============================================================================
# 🧹 MAINTENANCE COMMANDS
# =============================================================================

# Clean generated files and build artifacts
clean:
    @echo "🧹 Cleaning generated files and build artifacts..."
    rm -f {{project_name}}
    rm -f coverage.out coverage.html
    rm -rf examples/*.go examples/dev
    rm -rf result result-*
    go clean -cache
    @echo "✅ Cleanup complete"

# Format all code
format:
    @echo "📝 Formatting code..."
    go fmt ./...
    gofumpt -w . || echo "gofumpt not available, using go fmt"
    nixpkgs-fmt flake.nix .nix/*.nix || echo "nixpkgs-fmt not available"
    @echo "✅ Code formatted"

# =============================================================================
# 📊 PROJECT STATUS & INFO
# =============================================================================

# Show project status and metrics
status:
    @echo "📊 Devcmd Project Status"
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @echo "Grammar files: $(find {{grammar_dir}} -name '*.g4' | wc -l)"
    @echo "Generated files: $(find {{gen_dir}} -name '*.go' 2>/dev/null | wc -l || echo 0)"
    @echo "Go source files: $(find . -name '*.go' -not -path './{{gen_dir}}/*' | wc -l)"
    @echo "Test files: $(find . -name '*_test.go' | wc -l)"
    @echo "Nix files: $(find . -name '*.nix' | wc -l)"
    @echo "Total lines of code: $(find . -name '*.go' -not -path './{{gen_dir}}/*' -exec wc -l {} + | tail -1 | awk '{print $1}' || echo 0)"
    @echo ""
    @echo "Recent commits:"
    @git log --oneline -5 || echo "Not a git repository"

# =============================================================================
# 🔄 DEVELOPMENT WORKFLOWS
# =============================================================================

# Complete development workflow
workflow-dev:
    @echo "🔄 Running complete development workflow..."
    just setup
    just test
    just lint
    @echo "✅ Development workflow complete!"

# Release preparation workflow
workflow-release:
    @echo "📦 Running release preparation workflow..."
    just clean
    just setup
    just test-all
    just lint
    just nix-check
    just format
    @echo "✅ Ready for release!"

# Quick validation workflow
workflow-quick:
    @echo "⚡ Running quick validation..."
    just test-parser
    just test-generator
    just lint
    @echo "✅ Quick validation complete!"

# =============================================================================
# 🔧 ALIASES FOR CONVENIENCE
# =============================================================================

alias g := grammar
alias t := test
alias b := build
alias c := clean
alias f := format
alias l := lint
alias s := status

# Nix aliases
alias nb := nix-build
alias ne := nix-examples
alias nt := nix-test
alias nc := nix-check
