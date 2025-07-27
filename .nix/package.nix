# Package definition for devcmd
{ pkgs, lib, version ? "dev" }:

pkgs.stdenv.mkDerivation rec {
  pname = "devcmd";
  inherit version;

  src = ./..;

  nativeBuildInputs = with pkgs; [ go ];

  # Simple build without network requirements for now
  buildPhase = ''
    runHook preBuild
    
    echo "🔨 Building devcmd from multi-module workspace..."
    echo "⚠️  Note: This build may require network access for Go modules"
    
    # Set up Go build environment
    export GOCACHE=$TMPDIR/go-cache
    export GOPATH=$TMPDIR/go
    export CGO_ENABLED=0
    export GOPROXY=direct
    export GOSUMDB=off
    
    # Try building directly - Go will handle module resolution
    cd cli
    go build -o ../devcmd \
      -ldflags="-s -w -X main.Version=${version} -X main.BuildTime=1970-01-01T00:00:00Z" \
      ./main.go || {
        echo "❌ Build failed - likely due to missing dependencies"
        echo "💡 For local builds, use: nix develop && ./devcmd run build"
        exit 1
      }
    
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
