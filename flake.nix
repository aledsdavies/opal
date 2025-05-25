{
  description = "devcmd - Domain-specific language for generating development command CLIs";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          lib = nixpkgs.lib;

          # Import modular components
          devShell = import ./.nix/development.nix { inherit pkgs; };
          packageDef = import ./.nix/package.nix { inherit pkgs lib; version = "0.2.0"; };
          devcmdLib = import ./.nix/lib.nix { inherit pkgs self system lib; };
          tests = import ./.nix/tests.nix { inherit pkgs lib self system; };
          examples = import ./.nix/examples.nix { inherit pkgs lib self system; };

        in
        {
          # Main package
          packages = {
            default = packageDef;
            devcmd = packageDef;

            # Example CLIs
            inherit (examples.examples) basicDev webDev goProject rustProject dataScienceProject devOpsProject;

            # Test runner
            tests = tests.runAllTests;
            test-examples = examples.testExamples;
          };

          # Development shells
          devShells = {
            default = devShell;

            # Example development environments
            inherit (examples.shells) basicShell webShell goShell dataShell;

            # Test environment with all examples
            testEnv = pkgs.mkShell {
              name = "devcmd-test-env";
              buildInputs = with pkgs; [
                # All example CLIs
                self.packages.${system}.basicDev
                self.packages.${system}.webDev
                self.packages.${system}.goProject
                self.packages.${system}.rustProject
                self.packages.${system}.dataScienceProject
                self.packages.${system}.devOpsProject

                # Testing tools
                bash
                coreutils
                findutils
              ];

              shellHook = ''
                echo "🧪 Devcmd Test Environment"
                echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                echo ""
                echo "Available example CLIs:"
                echo "  dev          - Basic development commands"
                echo "  webdev       - Web development (frontend/backend)"
                echo "  godev        - Go project development"
                echo "  rustdev      - Rust project development"
                echo "  datadev      - Data science / Python development"
                echo "  devops       - DevOps / Infrastructure management"
                echo ""
                echo "Try: <cli-name> --help"
                echo ""
              '';
            };
          };

          # Library functions for other flakes to use
          lib = devcmdLib // {
            # Re-export utility functions
            inherit (devcmdLib.utils) getVersion postProcessors preProcessors;

            # Convenience functions
            mkBasicCLI = commandsContent: devcmdLib.mkDevCLI {
              inherit commandsContent;
              name = "devcmd-cli";
            };

            mkProjectCLI = { name, commands }: devcmdLib.mkDevCLI {
              inherit name;
              commandsContent = commands;
              validateCommands = true;
            };
          };

          # Checks (run with `nix flake check`)
          checks = {
            # Package builds
            package-builds = self.packages.${system}.default;

            # All tests pass
            tests-pass = tests.runAllTests;

            # Examples work
            examples-work = examples.testExamples;

            # Generated CLIs are valid
            cli-validation = pkgs.runCommand "validate-generated-clis"
              {
                nativeBuildInputs = [ pkgs.bash ] ++ (builtins.attrValues examples.examples);
              } ''
              echo "Validating all generated CLIs..."

              # Test that each CLI can show help
              ${lib.concatMapStringsSep "\n" (name: cli: ''
                echo "Validating ${name}..."
                ${cli.meta.mainProgram or name} --help >/dev/null
              '') (lib.mapAttrsToList (name: cli: cli) examples.examples)}

              echo "✅ All CLIs validated"
              touch $out
            '';
          };

          # Apps for easy running
          apps = {
            default = {
              type = "app";
              program = "${self.packages.${system}.default}/bin/devcmd";
            };

            # Example apps
            basicDev = {
              type = "app";
              program = "${self.packages.${system}.basicDev}/bin/dev";
            };

            webDev = {
              type = "app";
              program = "${self.packages.${system}.webDev}/bin/webdev";
            };

            goProject = {
              type = "app";
              program = "${self.packages.${system}.goProject}/bin/godev";
            };
          };

          # Formatter
          formatter = pkgs.nixpkgs-fmt;
        }) // {
      # Templates for other projects
      templates = {
        default = {
          path = ./templates/basic;
          description = "Basic project with devcmd CLI";
        };
      };

      # Overlay for use in other flakes
      overlays.default = final: prev: {
        devcmd = self.packages.${prev.system}.default;

        # Add library functions to pkgs
        devcmdLib = self.lib;
      };
    };
}
