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

          # Main devcmd package
          devcmdPackage = import ./.nix/package.nix { inherit pkgs lib; version = "0.2.0"; };

          # Library functions
          devcmdLib = import ./.nix/lib.nix { inherit pkgs self system lib; };

          # Simple examples (using correct template path)
          basicExample = devcmdLib.mkDevCLI {
            name = "dev";
            commandsFile = ./template/basic/commands.cli;
          };

        in
        {
          # Main package
          packages = {
            default = devcmdPackage;
            devcmd = devcmdPackage;

            # Example CLI
            basicDev = basicExample;
          };

          # Development shell (using existing development.nix)
          devShells.default = import ./.nix/development.nix { inherit pkgs; };

          # Library functions for other flakes
          lib = devcmdLib;

          # Apps for easy running
          apps = {
            default = {
              type = "app";
              program = "${self.packages.${system}.default}/bin/devcmd";
            };

            basicDev = {
              type = "app";
              program = "${self.packages.${system}.basicDev}/bin/dev";
            };
          };

          # Checks
          checks = {
            package-builds = self.packages.${system}.default;
            example-builds = self.packages.${system}.basicDev;
          };

          # Formatter
          formatter = pkgs.nixpkgs-fmt;
        }) // {

      # Templates for other projects
      templates = {
        default = {
          path = ./template/basic;
          description = "Basic project with devcmd CLI";
        };
      };

      # Overlay for use in other flakes
      overlays.default = final: prev: {
        devcmd = self.packages.${prev.system}.default;
        devcmdLib = self.lib.${prev.system};
      };
    };
}
