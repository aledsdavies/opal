# devcmd

A lightweight, extensible **D**eclarative **E**xecution **V**ocabulary for defining and orchestrating commands across environments, with seamless Nix integration.

## Overview

devcmd simplifies the creation and management of commands in development environments. Define commands and variables in a clean, maintainable syntax, and devcmd integrates them directly into your workflow.

```
# commands   <-- This is the default filename (no extension)
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

- Simple, declarative syntax for defining commands and workflows
- Variable definitions with substitution using `$(name)` syntax
- Block commands for multi-step processes
- Background process management with `watch` and `stop`
- Command continuations with backslash for improved readability
- Nix store path variable substitution for easy integration with flake-based environments
- No external dependencies required
- Extensible architecture for customization

## Installation

Add devcmd to your flake inputs:

```nix
# flake.nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
    devcmd.url = "github:aledsdavies/devcmd";
  };

  # Your outputs...
}
```

## Usage

1. Create a file named `commands` in your project root with your command definitions

2. Add devcmd to your flake and integrate it in your devShell:

```nix
{
  description = "Project with devcmd integration";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
    devcmd.url = "github:aledsdavies/devcmd";
  };

  outputs = { self, nixpkgs, devcmd }:
    let
      system = builtins.currentSystem;
      pkgs = nixpkgs.legacyPackages.${system};
    in
    {
      devShells.${system}.default = pkgs.mkShell {
        # Add your development dependencies
        buildInputs = with pkgs; [
          go
          nodejs
          # Add other tools as needed
        ];

        # Set up devcmd
        shellHook = (devcmd.lib.mkDevCommands {
          inherit pkgs system;
          # Optional: specify an alternate commands file location
          # commandsFile = ./path/to/commands;
        }).shellHook;
      };
    };
}
```

3. Enter your development environment:

```bash
$ nix develop
Dev shell initialized with your custom commands!
Available commands: build, run, watch-dev, stop-dev, deploy

$ run
# Executes: $(go) run ./cmd/main.go
```

This creates a fully configured development environment with your devcmd commands available as soon as you enter the shell.

## Command Files

devcmd will look for commands in the following order:

1. The path specified in `commandsFile` parameter
2. A file named `commands` (no extension) in your project root (default)
3. A file named `commands.txt`
4. A file named `commands.devcmd`

The recommended approach is to use a file named `commands` in your project root.

## Command Syntax

The command file uses a simple syntax:

```
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

# Define corresponding stop commands
stop <command-name>: <command-to-stop-processes>

# Comments start with #
```

Variables are referenced using `$(name)` syntax within commands.

Example:

```
# Define tools with full Nix store paths
def go = ${pkgs.go}/bin/go
def node = ${pkgs.nodejs}/bin/node
def python = ${pkgs.python3}/bin/python3

# Define project variables
def SRC_DIR = ./src
def OUT_DIR = ./dist

# Define commands
build: $(go) build -o $(OUT_DIR)/app $(SRC_DIR)/main.go
run: $(OUT_DIR)/app
test: $(go) test ./...
lint: $(go) vet ./...

# Web development commands
watch dev: {
  $(node) $(SRC_DIR)/scripts/dev-server.js;
  $(python) -m http.server 8080 --directory $(OUT_DIR) &
}

stop dev: pkill -f "http.server 8080"

# Compound commands
all: {
  build;
  test;
  lint
}
```

## Just Use Nix

While dev containers and Docker offer isolated environments, Nix provides several key advantages:

- **Truly reproducible environments**: Nix's content-addressed store ensures that every dependency is precisely tracked and can be perfectly reproduced.
- **Better resource efficiency**: No VM or container overhead - just the exact tools you need.
- **Cross-platform consistency**: The same Nix code works identically on Linux and macOS.
- **Incremental activation**: Nix environments can be entered instantly, no need to rebuild entire images.
- **Composable system**: Mix and match environments seamlessly, something containers can't easily do.

devcmd makes Nix development more approachable by simplifying the most common task: running commands in your project. No need to write complex shell hooks or remember esoteric Nix syntax - just define your commands once and they're available across all your development sessions.

## Extension Points

devcmd is designed to be extended. Key extension points include:

- Custom parser implementation in `pkg/parser/`
- Shell script generation templates
- Command metadata extraction

See the source code for detailed extension documentation.

## Maintenance Status

devcmd was created to solve my own development workflow challenges. While released under Apache 2.0 for anyone to use, it comes with no maintenance guarantees.

The project's design prioritizes:
- A focused, minimal core that does one thing well
- Clear extension points for customization
- Well-documented, easily forkable code

I'll review issues and PRs as time permits, but response times will vary. Extensions and forks are encouraged over feature requests. If you need something different, the modular codebase should make it easy to adapt to your needs.

This tool exists because it makes my Nix workflow better - hopefully it helps yours too.

## Beyond Development: Other Applications

While devcmd was initially designed for development environments, its declarative syntax makes it suitable for a wide range of automation scenarios:

- **Infrastructure Management**: Define cloud resources, orchestrate deployments
- **Data Pipelines**: Create ETL workflows, automate data processing
- **System Administration**: Automate maintenance tasks, monitor system health
- **Scientific Computing**: Orchestrate research workflows, manage computational pipelines

## License

This project is licensed under the Apache License, Version 2.0. See the LICENSE file for details.
