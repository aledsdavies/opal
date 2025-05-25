# Library functions for generating CLI packages from devcmd files
{ pkgs, self, system, lib }:

rec {
  # Generate a CLI package from devcmd commands
  mkDevCLI =
    {
      # Package name for the generated CLI
      name ? "devcmd-cli"

      # Source of commands - exactly one must be provided
    , commandsFile ? null      # Path to .devcmd file
    , commandsContent ? null   # Raw devcmd content as string
    , commandsDerivation ? null # Derivation that produces a .devcmd file

      # Processing and customization
    , preProcess ? (text: text)    # Function to transform input before parsing
    , postProcess ? (text: text)   # Function to transform generated Go before building
    , templateFile ? null          # Custom Go template file

      # Build options
    , version ? "generated"
    , meta ? { }
    , buildInputs ? [ ]
    , extraLdflags ? [ ]

      # Validation options
    , validateCommands ? true      # Whether to validate generated CLI
    , runTests ? false            # Whether to run tests on generated CLI
    }:

    let
      # Input validation
      commandSources = lib.count (x: x != null) [ commandsFile commandsContent commandsDerivation ];

      # Helper to read files safely
      safeReadFile = path:
        if builtins.pathExists path
        then builtins.readFile path
        else throw "File not found: ${toString path}";

      # Get commands content based on input type
      rawContent =
        if commandsFile != null then safeReadFile commandsFile
        else if commandsContent != null then commandsContent
        else if commandsDerivation != null then builtins.readFile "${commandsDerivation}/commands.devcmd"
        else throw "One of commandsFile, commandsContent, or commandsDerivation must be provided";

      # Process content
      processedContent = preProcess rawContent;

      # Write processed content to store
      commandsInput = pkgs.writeText "${name}-commands.devcmd" processedContent;

      # Get devcmd binary
      devcmdBin = self.packages.${system}.default;

      # Template arguments
      templateArgs = lib.optionalString (templateFile != null) "--template ${toString templateFile}";

      # Generate Go source
      goSource = pkgs.runCommand "${name}-go-source"
        {
          nativeBuildInputs = [ devcmdBin ];
          meta.description = "Generated Go source for ${name}";
        } ''
              mkdir -p $out

              echo "Generating Go CLI from devcmd file..."
              ${devcmdBin}/bin/devcmd generate ${templateArgs} ${commandsInput} > $out/main.go

              # Post-process generated Go if requested
              ${lib.optionalString (postProcess != (text: text)) ''
                echo "Post-processing generated Go code..."
                cp $out/main.go $out/main.go.orig
                cat $out/main.go.orig | ${postProcess} > $out/main.go
              ''}

              # Create go.mod
              cat > $out/go.mod << EOF
        module ${name}
        go 1.21
        EOF

              # Validate generated Go syntax
              ${lib.optionalString validateCommands ''
                echo "Validating generated Go code..."
                ${pkgs.go}/bin/go mod tidy -C $out
                ${pkgs.go}/bin/go build -C $out -o /dev/null ./...
                echo "✅ Generated Go code is valid"
              ''}
      '';

      # Build the CLI
      cli = pkgs.buildGoModule {
        pname = name;
        inherit version;
        src = goSource;
        vendorHash = null;

        inherit buildInputs;

        # Enhanced build flags
        ldflags = [
          "-s"
          "-w"
          "-X main.Version=${version}"
          "-X main.GeneratedBy=devcmd"
          "-X main.BuildTime=1970-01-01T00:00:00Z"
        ] ++ extraLdflags;

        # Optional testing
        doCheck = runTests;

        # Rename binary to match package name
        postInstall = ''
          if [ "$pname" != "$(basename $out/bin/*)" ]; then
            mv $out/bin/* $out/bin/${name}
          fi
        '';

        meta = {
          description = "Generated CLI from devcmd: ${name}";
          license = lib.licenses.mit;
          platforms = lib.platforms.unix;
          mainProgram = name;
        } // meta;
      };

    in
    if commandSources != 1
    then throw "Exactly one of commandsFile, commandsContent, or commandsDerivation must be provided (got ${toString commandSources})"
    else cli;

  # Create a development shell with generated CLI
  mkDevShell =
    { name ? "devcmd-shell"
    , cli ? null
    , extraPackages ? [ ]
    , shellHook ? ""
    }:

    pkgs.mkShell {
      inherit name;

      buildInputs = extraPackages ++ lib.optional (cli != null) cli;

      shellHook = ''
        echo "🚀 ${name} Development Shell"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

        ${lib.optionalString (cli != null) ''
          echo ""
          echo "Generated CLI available as: ${cli.meta.mainProgram or name}"
          echo ""
          echo "Available commands:"
          ${cli.meta.mainProgram or name} --help | grep -E "^  [a-zA-Z]" | head -10 || true
          if [ $(${cli.meta.mainProgram or name} --help | grep -E "^  [a-zA-Z]" | wc -l) -gt 10 ]; then
            echo "  ... (run '${cli.meta.mainProgram or name} --help' for full list)"
          fi
        ''}

        ${shellHook}
        echo ""
      '';
    };

  # Test a generated CLI with various scenarios
  testCLI =
    { cli
    , testScenarios ? [ ]  # List of { name, script } test scenarios
    , name ? cli.meta.mainProgram or "cli"
    }:

    pkgs.runCommand "${name}-tests"
      {
        nativeBuildInputs = [ cli pkgs.bash pkgs.coreutils ];
        meta.description = "Test scenarios for ${name} CLI";
      }
      (''
        mkdir -p $out

        echo "Testing CLI: ${name}"
        echo "===================="

        # Basic functionality test
        echo "Testing help command..."
        ${name} --help > $out/help.txt

        echo "Testing version command..."
        ${name} --version > $out/version.txt 2>&1 || echo "No version command" > $out/version.txt

      '' + lib.concatMapStrings
        (scenario: ''
          echo ""
          echo "Running test: ${scenario.name}"
          echo "-----------------------------"
          (
            ${scenario.script}
          ) > $out/${scenario.name}.log 2>&1
          if [ $? -eq 0 ]; then
            echo "✅ ${scenario.name} passed"
          else
            echo "❌ ${scenario.name} failed"
            cat $out/${scenario.name}.log
            exit 1
          fi
        '')
        testScenarios + ''

    echo ""
    echo "All tests passed! 🎉"
    echo "$(date)" > $out/success
  '');

  # Generate multiple CLIs from a directory of .devcmd files
  mkMultipleCLIs =
    { sourceDir
    , namePrefix ? ""
    , commonOptions ? { }
    }:

    let
      devcmdFiles = lib.filterAttrs
        (name: type:
          type == "regular" && lib.hasSuffix ".devcmd" name
        )
        (builtins.readDir sourceDir);

      cliName = filename:
        let baseName = lib.removeSuffix ".devcmd" filename;
        in "${namePrefix}${baseName}";

    in
    lib.mapAttrs
      (filename: _:
      mkDevCLI ({
        name = cliName filename;
        commandsFile = sourceDir + "/${filename}";
      } // commonOptions)
      )
      devcmdFiles;

  # Utilities for common patterns
  utils = {
    # Extract version from git or fallback
    getVersion = src:
      if src ? rev
      then "git-${builtins.substring 0 7 src.rev}"
      else "unknown";

    # Common post-processors
    postProcessors = {
      # Add custom imports
      addImports = imports: goCode:
        lib.replaceStrings
          [ "import (" ]
          [ "import (\n${lib.concatMapStringsSep "\n" (imp: "  \"${imp}\"") imports}" ]
          goCode;

      # Replace package name
      replacePackage = newName: goCode:
        lib.replaceStrings [ "package main" ] [ "package ${newName}" ] goCode;
    };

    # Common pre-processors
    preProcessors = {
      # Add common definitions
      addCommonDefs = defs: content:
        (lib.concatMapStringsSep "\n" (def: "def ${def.name} = ${def.value};") defs) + "\n\n" + content;

      # Filter out comments
      stripComments = content:
        lib.concatStringsSep "\n"
          (lib.filter (line: !lib.hasPrefix "#" (lib.trim line))
            (lib.splitString "\n" content));
    };
  };
}
