# Package definition for devcmd
{ pkgs, lib, version ? "dev" }:

pkgs.stdenv.mkDerivation rec {
  pname = "devcmd";
  inherit version;

  src = ./..;

  nativeBuildInputs = with pkgs; [ go ];

  buildPhase = ''
    runHook preBuild
    
    echo "🔨 Building devcmd from multi-module workspace..."
    
    # Set up Go build environment
    export GOCACHE=$TMPDIR/go-cache
    export GOPATH=$TMPDIR/go
    export CGO_ENABLED=0
    
    # Build the CLI module (it will use local modules via replace directives)
    cd cli
    go build -o ../devcmd \
      -ldflags="-s -w -X main.Version=${version} -X main.BuildTime=1970-01-01T00:00:00Z" \
      ./main.go
    
    echo "✅ devcmd build completed successfully"
    
    runHook postBuild
  '';

  installPhase = ''
    runHook preInstall
    
    mkdir -p $out/bin
    cp devcmd $out/bin/
    
    runHook postInstall
  '';

  doCheck = true;

  checkPhase = ''
    runHook preCheck
    
    echo "🧪 Running devcmd tests across all modules..."
    
    # Set up test environment
    export GOCACHE=$TMPDIR/go-cache
    export GOPATH=$TMPDIR/go
    
    # Test each module individually (avoiding workspace issues)
    for module in core runtime cli; do
      echo "Testing module: $module"
      cd $module
      go test -v ./...
      cd ..
    done
    
    runHook postCheck
  '';

  meta = with lib; {
    description = "Domain-specific language for generating development command CLIs";
    homepage = "https://github.com/aledsdavies/devcmd";
    license = licenses.mit;
    maintainers = [ maintainers.aledsdavies ];
    platforms = platforms.unix;
    mainProgram = "devcmd";
  };
}
