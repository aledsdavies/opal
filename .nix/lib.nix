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
  # Generate a CLI package from devcmd commands using stdenv.mkDerivation for better control
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

      # Get devcmd binary
      devcmdBin =
        if self != null then self.packages.${system}.devcmd or self.packages.${system}.default
        else throw "Self reference required for CLI generation. Cannot build '${name}' without devcmd parser.";

      # Build the CLI using a fixed-output derivation for network access
      # This allows Go module downloads while maintaining reproducibility
      cliPackage = pkgs.stdenv.mkDerivation {
        pname = name;
        inherit version;
        
        # Use the commands file as source
        src = commandsSrc;
        
        nativeBuildInputs = [ devcmdBin pkgs.go pkgs.cacert ];
        
        # Don't unpack the source - we'll use it directly
        dontUnpack = true;
        
        buildPhase = ''
          # Set up Go environment
          export HOME=$TMPDIR
          export GOCACHE=$TMPDIR/go-cache
          export GOMODCACHE=$TMPDIR/go-mod-cache
          export GOPATH=$TMPDIR/go
          mkdir -p $GOCACHE $GOMODCACHE $GOPATH
          
          # Generate and build the CLI binary (with network access for modules)
          ${devcmdBin}/bin/devcmd build \
            --file "$src" \
            --binary "${binaryName}" \
            -o "${binaryName}"
        '';
        
        installPhase = ''
          mkdir -p $out/bin
          cp ${binaryName} $out/bin/
          chmod +x $out/bin/${binaryName}
        '';
        
        # Fixed-output derivation attributes for network access
        outputHashAlgo = "sha256";
        outputHashMode = "recursive";
        # Use fake hash initially - Nix will tell us the real hash on first build
        outputHash = lib.fakeSha256;
        
        meta = with lib; {
          description = "Generated CLI from devcmd (fixed-output derivation)";
          license = licenses.mit;
          platforms = platforms.unix;
          mainProgram = binaryName;
        } // meta;
      };

    in
    cliPackage;
}
