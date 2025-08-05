# Library functions for generating CLI packages from devcmd files with proper sandbox support
{ pkgs, self, lib, gitRev, system }:

let
  # Helper function to safely read files
  tryReadFile = path:
    if builtins.pathExists (toString path) then
      builtins.readFile path
    else null;

in
rec {
  # Generate a CLI package from devcmd commands
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
            ./commands.cli  # Look in current directory
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

      # Create the commands file as a source
      commandsSrc = pkgs.writeText "${name}-commands.cli" processedContent;

      # All-in-one FOD: Build devcmd from source and generate CLI in single derivation
      # TODO: Migrate to dynamic derivations when they become stable for cleaner separation
      # This avoids store path references while maintaining network access for Go modules
      cliPackage = pkgs.stdenv.mkDerivation {
        pname = name;
        inherit version;
        
        # We're in the project root, just use the current directory
        src = ./.;
        
        nativeBuildInputs = with pkgs; [ go cacert ];
        
        buildPhase = ''
          # Set up Go environment for sandbox
          export HOME=$TMPDIR
          export GOCACHE=$TMPDIR/go-cache
          export GOMODCACHE=$TMPDIR/go-mod-cache
          export GOPATH=$TMPDIR/go
          mkdir -p $GOCACHE $GOMODCACHE $GOPATH
          
          # We already have all the source files, just add the commands file
          cat > commands.cli <<'EOF'
          ${processedContent}
          EOF
          
          # Build devcmd from current source
          echo "Building devcmd from source..."
          cd cli
          GOWORK=off go build -o ../devcmd ./main.go
          cd ..
          chmod +x ./devcmd
          
          # Generate and build CLI using devcmd
          echo "Generating CLI with devcmd..."
          ./devcmd build --file commands.cli --binary "${binaryName}" -o "${binaryName}"
        '';
        
        installPhase = ''
          mkdir -p $out/bin
          cp ${binaryName} $out/bin/
          chmod +x $out/bin/${binaryName}
        '';
        
        # Fixed-output derivation for network access during Go module downloads
        outputHashAlgo = "sha256";
        outputHashMode = "recursive";
        outputHash = lib.fakeSha256; # Will be updated on first build
        
        meta = with lib; {
          description = "Generated CLI from devcmd (all-in-one FOD)";
          license = licenses.mit;
          platforms = platforms.unix;
          mainProgram = binaryName;
        } // meta;
      };

    in
    cliPackage;
}
