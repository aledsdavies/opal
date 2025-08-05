# Library functions for generating CLI packages using shell scripts
{ pkgs, self, lib, gitRev, system }:

let
  # Helper function to safely read files
  tryReadFile = path:
    if builtins.pathExists (toString path) then
      builtins.readFile path
    else null;

in
rec {
  # Generate a shell script that runs devcmd build for CLI generation
  mkDevCLI =
    {
      # Package name 
      name

      # Binary name (defaults to "dev" if not specified)
    , binaryName ? "dev"

      # Content sources
    , commandsFile ? null
    , commandsContent ? null

      # Processing and build options
    , preProcess ? (text: text)
    , version ? "generated"
    , meta ? { }
    }:

    let
      # Content resolution logic
      fileContent =
        if commandsFile != null then tryReadFile commandsFile
        else null;

      inlineContent =
        if commandsContent != null then commandsContent
        else null;

      # Auto-detect with commands.cli as default
      autoDetectContent =
        let
          candidates = [
            ../commands.cli # Look in parent directory (project root)
            ./commands.cli # Look in current directory
            ./.commands.cli # Hidden variant
          ];

          findFirst = paths:
            if paths == [ ] then null
            else
              let candidate = builtins.head paths;
              in
              if builtins.pathExists (toString candidate) then tryReadFile candidate
              else findFirst (builtins.tail paths);
        in
        findFirst candidates;

      finalContent =
        if fileContent != null then fileContent
        else if inlineContent != null then inlineContent
        else if autoDetectContent != null then autoDetectContent
        else throw "No commands content found for CLI '${name}'. Expected commands.cli file or explicit content.";

      processedContent = preProcess finalContent;

      # Get devcmd binary
      devcmdBin =
        if self != null then self.packages.${system}.devcmd or self.packages.${system}.default
        else throw "Self reference required for CLI generation. Cannot build '${name}' without devcmd parser.";

      # Create a shell script that generates the CLI
      cliScript = pkgs.writeShellScriptBin binaryName ''
        #!/usr/bin/env bash
        set -euo pipefail
        
        # Create temporary commands file
        TEMP_COMMANDS=$(mktemp -t commands-XXXXXX.cli)
        trap "rm -f $TEMP_COMMANDS" EXIT
        
        cat > "$TEMP_COMMANDS" <<'EOF'
        ${processedContent}
        EOF
        
        # Check if compiled binary already exists and is current
        BINARY_PATH="./${binaryName}-compiled"
        if [[ -f "$BINARY_PATH" ]]; then
          echo "✅ ${binaryName} CLI ready"
          exec "$BINARY_PATH" "$@"
        fi
        
        # Generate the CLI binary
        echo "🔨 Generating ${binaryName} CLI..."
        ${devcmdBin}/bin/devcmd build --file "$TEMP_COMMANDS" --binary "${binaryName}" -o "$BINARY_PATH"
        
        # Execute the generated binary with all arguments
        exec "$BINARY_PATH" "$@"
      '';

    in
    cliScript;
}
