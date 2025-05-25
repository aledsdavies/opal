# Package definition for devcmd
{ pkgs, lib, version ? "0.2.0" }:

pkgs.buildGoModule rec {
  pname = "devcmd";
  inherit version;

  src = lib.cleanSource ../.;

  # Go module hash - update when dependencies change
  # Set to null initially, then update with the hash from build error
  vendorHash = null;

  # Build only the main CLI
  subPackages = [ "cmd/devcmd" ];

  # Pre-build steps
  preBuild = ''
    # Ensure ANTLR generated files exist
    if [ ! -f internal/gen/devcmd_lexer.go ]; then
      echo "ANTLR generated files missing. Please run 'just grammar' first."
      exit 1
    fi
  '';

  # Build flags
  ldflags = [
    "-s"
    "-w"
    "-X main.Version=${version}"
    "-X main.GitCommit=${src.rev or "unknown"}"
    "-X main.BuildTime=1970-01-01T00:00:00Z"
  ];

  # Tests
  checkPhase = ''
    go test -v ./pkgs/parser
    go test -v ./pkgs/generator
  '';

  # Install additional files
  postInstall = ''
    # Install grammar files for reference
    mkdir -p $out/share/devcmd
    cp -r grammar $out/share/devcmd/

    # Install examples
    mkdir -p $out/share/devcmd/examples
    cp -r examples/*.devcmd $out/share/devcmd/examples/ 2>/dev/null || true

    # Install documentation
    mkdir -p $out/share/doc/devcmd
    cp README.md $out/share/doc/devcmd/ 2>/dev/null || true
    cp CODE_GUIDELINES.md $out/share/doc/devcmd/ 2>/dev/null || true
  '';

  meta = with lib; {
    description = "A domain-specific language for generating development command CLIs";
    longDescription = ''
      Devcmd is a DSL that allows you to define development commands, variables,
      and service management in a declarative way, then generates efficient
      Go CLI tools from those definitions.

      Features:
      - Variables and references: def SRC = ./src; build: cd $(SRC) && make
      - Service management: watch server: npm start; stop server: pkill node
      - POSIX shell syntax: check: (which go && echo "found") || exit 1
      - Block commands: setup: { npm install; go mod tidy; echo done }
      - Background processes: services: { server &; worker &; monitor }
    '';
    homepage = "https://github.com/aledsdavies/devcmd";
    license = licenses.mit;
    maintainers = [ maintainers.aledsdavies or "aledsdavies" ];
    platforms = platforms.unix;
    mainProgram = "devcmd";
  };
}
