# Package definition for devcmd CLI (built from cli module with Go workspace support)
{ pkgs, lib, version ? "dev" }:

pkgs.buildGoModule rec {
  pname = "devcmd";
  inherit version;

  src = ./..; # repo root that contains go.work
  modRoot = "cli"; # path to CLI module's go.mod
  subPackages = [ "." ]; # build the main package

  # Critical: Disable workspace mode and set up proper Go environment for Nix sandbox
  env.GOWORK = "off";
  env.GOCACHE = "$TMPDIR/go-cache";
  env.GOMODCACHE = "$TMPDIR/go-mod-cache";
  
  # Force GOWORK=off in vendor phase and set up sandbox-compatible paths
  overrideModAttrs = _: { 
    env.GOWORK = "off";
    env.GOCACHE = "$TMPDIR/go-cache";
    env.GOMODCACHE = "$TMPDIR/go-mod-cache";
  };

  # Vendor hash for CLI module dependencies  
  vendorHash = "sha256-5el+4EYvfYG6t9uBNpRCBrRfevgHYxFDMIxTltn1w18=";

  # Build with version info
  ldflags = [
    "-s"
    "-w"
    "-X main.Version=${version}"
    "-X main.BuildTime=1970-01-01T00:00:00Z"
  ];

  # Rename binary from 'cli' to 'devcmd'
  postInstall = ''
    mv $out/bin/cli $out/bin/devcmd
  '';

  doCheck = false; # Skip tests during build for now

  meta = with lib; {
    description = "Domain-specific language for generating development command CLIs";
    homepage = "https://github.com/aledsdavies/devcmd";
    license = licenses.mit;
    maintainers = [ maintainers.aledsdavies ];
    platforms = platforms.unix;
    mainProgram = "devcmd";
  };
}
