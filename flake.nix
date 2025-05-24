{
  description = "devcmd - Go CLI generator for development commands";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs, ... }:
    let
      lib = nixpkgs.lib;
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f: lib.genAttrs systems f;
      pkgsFor = system: import nixpkgs { inherit system; };
      version = "0.2.0";
    in
    {
      # Go parser binary package
      packages = forAllSystems (system:
        let pkgs = pkgsFor system;
        in {
          default = pkgs.buildGoModule {
            pname = "devcmd-parser";
            inherit version;
            src = ./.;
            vendorHash = null;
            subPackages = [ "cmd/devcmd-parser" ];

            meta = with lib; {
              description = "Parser and generator for devcmd DSL";
              license = licenses.mit;
            };
          };
        }
      );

      # Library function to generate CLI packages
      lib = {
        mkDevCLI =
          { pkgs
          , system ? builtins.currentSystem
          , commandsFile ? null
          , commandsContent ? null
          , preProcess ? (text: text)
          , templateFile ? null
          , name ? "devcmd"
          }:
          let
            # Helper to read files safely
            safeReadFile = path:
              if builtins.pathExists path
              then builtins.readFile path
              else null;

            # Get commands content
            finalContent =
              if commandsFile != null then safeReadFile commandsFile
              else if commandsContent != null then commandsContent
              else throw "Either commandsFile or commandsContent must be provided";

            # Process content
            processedContent = preProcess finalContent;
            processedPath = pkgs.writeText "commands-input" processedContent;

            # Parser binary
            parserBin = self.packages.${system}.default;

            # Template arguments
            templateArgs =
              if templateFile != null && builtins.pathExists templateFile
              then "--template ${toString templateFile}"
              else "";

            # Generate Go source
            goSource = pkgs.runCommand "${name}-go-source"
              { nativeBuildInputs = [ parserBin ]; }
              ''
                mkdir -p $out
                ${parserBin}/bin/devcmd-parser --format=go ${templateArgs} ${processedPath} > $out/main.go

                cat > $out/go.mod << 'EOF'
                module ${name}-cli
                go 1.21
                EOF
              '';

            # Compile the CLI
            cli = pkgs.buildGoModule {
              pname = "${name}-cli";
              version = "generated";
              src = goSource;
              vendorHash = null;

              # Override the binary name to match the desired command name
              postInstall = ''
                mv $out/bin/${name}-cli $out/bin/${name}
              '';

              meta = {
                description = "Generated ${name} CLI";
              };
            };

          in
          cli;
      };

      # Development shell for the project itself
      devShells = forAllSystems (system:
        let
          pkgs = pkgsFor system;
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go
              gopls
              go-tools
              antlr4
            ];

            shellHook = ''
              echo "devcmd development environment"
              echo ""
              echo "Available commands:"
              echo "  go run ./cmd/devcmd-parser --help"
              echo "  go build ./cmd/devcmd-parser"
              echo "  go test ./..."
              echo "  go generate ./..."
            '';
          };

          # Example shell with generated CLI
          example =
            let
              exampleCLI = self.lib.mkDevCLI {
                inherit pkgs system;
                name = "example";
                commandsContent = ''
                  # Example commands
                  def PORT = 8080;

                  build: go build -o bin/devcmd ./cmd/devcmd-parser
                  test: go test -v ./...

                  watch demo: {
                    echo "Starting demo server on port $(PORT)";
                    python3 -m http.server $(PORT) &
                  }

                  stop demo: pkill -f "python3 -m http.server"

                  clean: rm -rf bin/
                '';
              };
            in
            pkgs.mkShell {
              buildInputs = with pkgs; [
                go
                python3
                exampleCLI
              ];

              shellHook = ''
                echo "devcmd example environment"
                echo ""
                echo "Generated CLI available as: example"
                echo "  example build        # Build the parser"
                echo "  example test         # Run tests"
                echo "  example watch demo   # Start demo server"
                echo "  example status       # Show running processes"
                echo "  example logs demo    # View server logs"
                echo "  example stop demo    # Stop server"
              '';
            };
        }
      );

      # Project template
      templates.default = {
        path = ./template;
        description = "Project template with devcmd CLI generation";
      };

      # Formatter
      formatter = forAllSystems (system: (pkgsFor system).nixpkgs-fmt);
    };
}
