# devcmd testing and development commands

# Default recipe - show available commands
default:
    @just --list

# Run all tests
test-all: test-go test-parser test-generator test-nix

# === Go Tests ===

# Run Go unit tests
test-go:
    @echo "Running Go unit tests..."
    go test -v ./...

# Run Go tests with coverage
test-go-coverage:
    @echo "Running Go tests with coverage..."
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report: coverage.html"

# Build the parser binary
build:
    @echo "Building devcmd-parser..."
    go build -o bin/devcmd-parser ./cmd/devcmd-parser

# === Parser Tests ===

# Test parser with sample files
test-parser: build
    @echo "Testing parser with sample commands..."
    ./bin/devcmd-parser test/sample.devcmd

# Test shell generation
test-shell: build
    @echo "Testing shell generation..."
    ./bin/devcmd-parser --format=shell test/sample.devcmd > test/output.sh
    @echo "Generated shell script: test/output.sh"

# Test Go generation
test-go-gen: build
    @echo "Testing Go CLI generation..."
    ./bin/devcmd-parser --format=go test/sample.devcmd > test/output.go
    @echo "Generated Go code: test/output.go"

# === Generator Tests ===

# Test generator package directly
test-generator:
    @echo "Testing generator package..."
    go test -v ./pkgs/generator/...

# Generate and compile a test CLI
test-cli-generation: build
    @echo "Generating and compiling test CLI..."
    mkdir -p test/cli
    ./bin/devcmd-parser --format=go test/sample.devcmd > test/cli/main.go
    cd test/cli && echo "module testcli\ngo 1.21" > go.mod
    cd test/cli && go build -o testcli .
    @echo "Test CLI compiled: test/cli/testcli"

# Test the generated CLI
test-generated-cli: test-cli-generation
    @echo "Testing generated CLI..."
    cd test/cli && ./testcli help
    @echo "Try: cd test/cli && ./testcli status"

# === Nix Tests ===

# Test Nix flake
test-nix:
    @echo "Testing Nix flake..."
    nix flake check

# Test development shell
test-dev-shell:
    @echo "Testing development shell..."
    nix develop --command echo "Dev shell works"

# Test example shell with generated CLI
test-example-shell:
    @echo "Testing example shell..."
    nix develop .#example --command example help

# Build using Nix
build-nix:
    @echo "Building with Nix..."
    nix build

# === Sample File Creation ===

# Create sample test files
setup-tests:
    @echo "Creating test files..."
    mkdir -p test
    cat > test/sample.devcmd << 'EOF'
# Sample devcmd file for testing
def PORT = 8080;
def SRC = ./src;

# Basic commands
build: go build -o bin/app $(SRC)/main.go
test: go test -v ./...
clean: rm -rf bin/

# Watch commands
watch api: {
  echo "Starting API server on port $(PORT)";
  go run $(SRC)/main.go --port=$(PORT) &
}

watch frontend: {
  echo "Starting frontend dev server";
  npm run dev &
}

# Stop commands
stop api: pkill -f "go run.*main.go"
stop frontend: pkill -f "npm run dev"

# Complex block command
dev: {
  echo "Setting up development environment";
  go mod tidy;
  npm install;
  echo "Development environment ready"
}
EOF
    @echo "Created test/sample.devcmd"

# === Integration Tests ===

# Full integration test - create, build, and test CLI
test-integration: setup-tests build
    @echo "Running full integration test..."

    # Generate Go CLI
    ./bin/devcmd-parser --format=go test/sample.devcmd > test/integration.go

    # Create module and build
    cd test && echo "module integration\ngo 1.21" > go.mod
    cd test && go build -o testcli integration.go

    # Test basic commands
    cd test && ./testcli help
    @echo "✅ Help command works"

    cd test && ./testcli status
    @echo "✅ Status command works"

    @echo "Integration test completed successfully!"

# === Nix Integration Tests ===

# Test mkDevCLI function
test-nix-lib: setup-tests
    @echo "Testing Nix library function..."
    cat > test/test-flake.nix << 'EOF'
{
  inputs.devcmd.url = "path:.";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { devcmd, nixpkgs, ... }:
    let
      pkgs = nixpkgs.legacyPackages.x86_64-linux;

      testCLI = devcmd.lib.mkDevCLI {
        inherit pkgs;
        system = "x86_64-linux";
        name = "testcli";
        commandsFile = ./sample.devcmd;
      };
    in {
      packages.x86_64-linux.default = testCLI;
    };
}
EOF
    cd test && nix build --file test-flake.nix
    cd test && ./result/bin/testcli help
    @echo "✅ Nix library function works!"

# === Benchmarks ===

# Benchmark parser performance
benchmark: build setup-tests
    @echo "Benchmarking parser..."
    time ./bin/devcmd-parser --format=shell test/sample.devcmd > /dev/null
    time ./bin/devcmd-parser --format=go test/sample.devcmd > /dev/null

# === Cleanup ===

# Clean up test files and build artifacts
clean:
    @echo "Cleaning up..."
    rm -rf bin/
    rm -rf test/
    rm -f coverage.out coverage.html
    go clean

# Clean Nix build results
clean-nix:
    @echo "Cleaning Nix artifacts..."
    rm -rf result result-*

# Full cleanup
clean-all: clean clean-nix
    @echo "Full cleanup completed"

# === Development Helpers ===

# Watch for changes and run tests
watch:
    @echo "Watching for changes..."
    find . -name "*.go" -o -name "*.g4" | entr -c just test-go

# Format code
fmt:
    @echo "Formatting code..."
    go fmt ./...
    nix fmt

# Lint code
lint:
    @echo "Linting code..."
    golangci-lint run

# Generate ANTLR code
generate:
    @echo "Generating ANTLR code..."
    go generate ./...

# === Documentation ===

# Generate documentation
docs:
    @echo "Generating documentation..."
    go doc -all ./... > docs/api.txt
    @echo "API documentation: docs/api.txt"
