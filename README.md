# devcmd

A lightweight, extensible **D**eclarative **E**xecution **V**ocabulary for defining and orchestrating commands across environments, with seamless Nix integration.

## Overview

devcmd simplifies the creation and management of commands in development environments. Define commands and variables in a clean, maintainable syntax, and devcmd generates either shell functions or standalone CLI tools that integrate directly into your workflow.

```bash
# commands.cli   <-- Preferred extension (also supports 'commands' with no extension)
def go = ${pkgs.go}/bin/go
def npm = ${pkgs.nodejs}/bin/npm
def SRC_DIR = ./src

# Simple commands
build: $(go) build -o ./bin/app ./cmd/main.go
run: $(go) run ./cmd/main.go

# Block commands with multiple statements
watch dev: {
  $(npm) run build-css;
  $(go) run $(SRC_DIR)/cmd/main.go;
  $(npm) run watch-assets &
}

# Stop command for watch processes
stop dev: pkill -f "watch-assets"

# Command with continuation lines
deploy: aws s3 sync \
  ./dist \
  s3://my-bucket/app/ \
  --delete
```

## Features

- **Two Integration Modes**: Shell hook functions OR standalone CLI binaries
- Simple, declarative syntax for defining commands and workflows
- Variable definitions with substitution using `$(name)` syntax
- Block commands for multi-step processes
- **Smart watch/stop pairing**: Automatic subcommand generation (`mycli dev start/stop`)
- **Safe process management**: Graceful shutdown with PID tracking for orphaned watch commands
- Command continuations with backslash for improved readability
- Nix store path variable substitution for easy integration with flake-based environments
- **POSIX shell syntax**: Full support for parentheses, pipes, redirections
- Extensible architecture for customization

## Project Structure

```
devcmd/
├── cmd/devcmd/           # Main CLI entry point
├── pkgs/
│   ├── parser/           # ANTLR-based command parser
│   └── generator/        # Go CLI code generation
├── internal/gen/         # Generated ANTLR parser code
├── grammar/             # ANTLR grammar definition
├── .nix/               # Nix build configuration
│   ├── lib.nix         # Library functions
│   ├── package.nix     # Main package definition
│   ├── development.nix # Development shell
│   └── examples.nix    # Example configurations
├── template/basic/     # Project template
│   ├── commands.cli    # Example commands
│   ├── flake.nix      # Template flake
│   └── README.md      # Template documentation
└── flake.nix          # Main flake configuration
```

## Installation

Add devcmd to your flake inputs:

```nix
# flake.nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    devcmd.url = "github:aledsdavies/devcmd";
  };

  # Your outputs...
}
```

## Usage

### Option 1: Shell Hook Integration (Recommended for Development)

Create shell functions that are available in your `nix develop` environment:

```nix
{
  description = "Project with devcmd shell integration";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    devcmd.url = "github:aledsdavies/devcmd";
  };

  outputs = { self, nixpkgs, devcmd }:
    let
      system = builtins.currentSystem;
      pkgs = nixpkgs.legacyPackages.${system};
    in
    {
      devShells.${system}.default = pkgs.mkShell {
        buildInputs = with pkgs; [
          go nodejs python3  # Your development dependencies
        ];

        # Generate shell functions from commands.cli
        shellHook = (devcmd.lib.mkDevCommands {
          inherit pkgs system;
          # Optional: specify alternate commands file
          # commandsFile = ./path/to/commands.cli;
        }).shellHook;
      };
    };
}
```

Usage:
```bash
$ nix develop
🚀 devcmd commands loaded from auto-detected file
Available commands: build, run, dev-start, dev-stop, deploy

$ build        # Runs: go build -o ./bin/app ./cmd/main.go
$ dev start    # Starts development processes in background
$ dev stop     # Stops development processes
```

### Option 2: Standalone CLI Generation

Generate a standalone CLI binary that can be distributed and used anywhere:

```nix
{
  description = "Project with standalone devcmd CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    devcmd.url = "github:aledsdavies/devcmd";
  };

  outputs = { self, nixpkgs, devcmd }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};

      # Generate standalone CLI from commands.cli
      projectCLI = devcmd.lib.mkDevCLI {
        name = "myproject";
        commandsFile = ./commands.cli;
        version = "1.0.0";
      };
    in
    {
      packages.${system} = {
        default = projectCLI;
        cli = projectCLI;
      };

      # Development shell with the CLI available
      devShells.${system}.default = devcmd.lib.mkDevShell {
        name = "myproject-dev";
        cli = projectCLI;
        extraPackages = with pkgs; [ git curl ];
      };

      apps.${system}.default = {
        type = "app";
        program = "${projectCLI}/bin/myproject";
      };
    };
}
```

Usage:
```bash
$ nix build
$ ./result/bin/myproject --help
Available commands:
  build
  dev start|stop
  deploy
  status              - Show running background processes
  logs <process>      - Show logs for a background process

$ ./result/bin/myproject build
$ ./result/bin/myproject dev start
$ ./result/bin/myproject status
$ ./result/bin/myproject dev stop
```

## Command Files

devcmd will look for commands in the following order:

1. The path specified in `commandsFile` parameter
2. `./commands.cli` (preferred extension)
3. `./commands` (no extension)
4. Legacy `.devcmd` files for backward compatibility

## Command Syntax

### Basic Syntax

```bash
# Define variables (usually Nix store paths or project constants)
def <name> = <value>

# Define simple commands
<command-name>: <command-to-execute>

# Define block commands
<command-name>: {
  <command1>;
  <command2>;
  <command3> &  # Run in background with &
}

# Define watch commands with background processes
watch <command-name>: {
  <start-process-1>;
  <start-process-2>;
  <start-process-3> &
}

# Define corresponding stop commands (optional)
stop <command-name>: <command-to-stop-processes>

# Comments start with #
```

### Watch/Stop Command Pairing

devcmd intelligently groups watch/stop commands:

```bash
# Watch with custom stop
watch dev: npm start
stop dev: npm stop
# → Generates: mycli dev start|stop

# Watch without stop (uses safe PID management)
watch api: go run main.go
# → Generates: mycli api start|stop (stop uses graceful SIGTERM/SIGKILL)

# Regular commands remain unchanged
build: go build ./...
# → Generates: mycli build
```

### Advanced Examples

```bash
# Define tools with full Nix store paths
def go = ${pkgs.go}/bin/go
def node = ${pkgs.nodejs}/bin/node
def python = ${pkgs.python3}/bin/python3

# Define project variables
def SRC_DIR = ./src
def OUT_DIR = ./dist
def PORT = 8080

# Simple commands
build: $(go) build -o $(OUT_DIR)/app $(SRC_DIR)/main.go
run: $(OUT_DIR)/app
test: $(go) test ./...
lint: $(go) vet ./...

# POSIX shell syntax support
check-deps: (which $(go) && echo "Go found") || (echo "Go missing" && exit 1)

# Complex watch commands with multiple processes
watch full-stack: {
  echo "Starting full development stack...";
  (cd frontend && $(node) server.js --port=3000) &;
  (cd backend && $(go) run main.go --port=$(PORT)) &;
  $(python) -m http.server 8080 --directory $(OUT_DIR) &;
  echo "All services started on ports 3000, $(PORT), 8080"
}

# Custom stop with cleanup
stop full-stack: {
  echo "Stopping all services...";
  pkill -f "server.js";
  pkill -f "go run main.go";
  pkill -f "http.server 8080";
  echo "All services stopped"
}

# Command continuations for readability
deploy: aws s3 sync \
  $(OUT_DIR) \
  s3://my-bucket/app/ \
  --delete \
  --cache-control "max-age=3600"

# Compound commands
all: {
  check-deps;
  build;
  test;
  lint
}
```

## CLI Features

### Process Management

Generated CLIs include built-in process management for watch commands:

```bash
$ mycli status
NAME            PID      STATUS     STARTED              COMMAND
dev             12345    running    14:32:15             npm start
api             12346    running    14:32:20             go run main.go

$ mycli logs dev
[14:32:15] Starting development server...
[14:32:16] Server ready on port 3000

$ mycli dev stop
Stopping process dev (PID: 12345)...
Process dev stopped successfully
```

### Help and Discovery

```bash
$ mycli --help
Available commands:
  status              - Show running background processes
  logs <process>      - Show logs for a background process
  build
  test
  dev start|stop
  api start|stop
  deploy

$ mycli dev
Usage: mycli dev <start|stop>
```

## Quick Start Template

Create a new project with devcmd:

```bash
$ nix flake init -t github:aledsdavies/devcmd
$ ls
commands.cli  flake.nix  README.md

$ cat commands.cli
# Basic development commands template
def SRC = ./src
def BUILD_DIR = ./build

build: {
  echo "Building project..."
  mkdir -p $(BUILD_DIR)
  echo "Build complete"
}

test: {
  echo "Running tests..."
  echo "Tests complete"
}

watch dev: {
  echo "Starting development mode..."
  echo "Development mode started"
}

stop dev: {
  echo "Stopping development processes..."
  echo "Development processes stopped"
}

$ nix develop
🚀 myproject-dev Development Shell
Generated CLI available as: myproject
Run 'myproject --help' to see available commands

$ myproject build
Building project...
Build complete
```

## Library API

### `mkDevCommands`

Generates shell functions for development environments:

```nix
devcmd.lib.mkDevCommands {
  inherit pkgs system;
  commandsFile = ./path/to/commands.cli;  # Optional
  commandsContent = "build: echo hello";  # Optional inline
  preProcess = text: "# Header\n" + text; # Optional preprocessing
  debug = true;                           # Optional debug output
}
```

### `mkDevCLI`

Generates standalone CLI binaries:

```nix
devcmd.lib.mkDevCLI {
  name = "mycli";
  commandsFile = ./commands.cli;
  version = "1.0.0";
  meta = { description = "My project CLI"; };
}
```

### `mkDevShell`

Creates development shells with integrated CLIs:

```nix
devcmd.lib.mkDevShell {
  name = "myproject-dev";
  cli = myGeneratedCLI;
  extraPackages = with pkgs; [ git curl ];
  shellHook = "echo Welcome!";
}
```

## Architecture

devcmd consists of several key components:

- **ANTLR Grammar**: Robust parsing of command syntax with POSIX shell support
- **Go Code Generator**: Template-based generation of standalone CLI binaries
- **Nix Integration**: Seamless integration with Nix development environments
- **Process Management**: Safe background process handling with PID tracking
- **Command Grouping**: Intelligent pairing of watch/stop commands

## Why Nix?

While dev containers and Docker offer isolated environments, Nix provides several key advantages:

- **Truly reproducible environments**: Content-addressed store ensures perfect reproducibility
- **Better resource efficiency**: No VM or container overhead
- **Cross-platform consistency**: Same code works identically on Linux and macOS
- **Incremental activation**: Instant environment activation, no rebuilding
- **Composable system**: Mix and match environments seamlessly

devcmd makes Nix development more approachable by simplifying command orchestration without complex shell scripting.

## Contributing

devcmd follows the [CODE_GUIDELINES.md](CODE_GUIDELINES.md) for development practices:

- **Safety & Fail-Fast**: Comprehensive error handling and validation
- **Precision & Determinism**: Reproducible builds and deterministic execution
- **Developer Experience**: Clear naming and minimal abstractions

See the project structure above for key directories and the justfile for development commands.

## License

This project is licensed under the Apache License, Version 2.0. See the LICENSE file for details.
