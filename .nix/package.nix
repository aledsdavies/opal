# Package definition for devcmd CLI (built from cli module)
{ pkgs, lib, version ? "dev" }:

pkgs.buildGoModule rec {
  pname = "devcmd";
  inherit version;

  src = ./..;  # repo root that contains go.work
  modRoot = "cli";  # path to CLI module's go.mod
  subPackages = [ "." ];  # build the main package

  # Turn off workspace mode for vendoring (required since Go 1.22)
  env.GOWORK = "off";
  env.GOCACHE = "/tmp/go-cache";
  env.GOMODCACHE = "/tmp/go-mod-cache";

  # Vendor hash for Go module dependencies
  vendorHash = "sha256-fxcLJ9rqMBwVWMK19FHa6dlOe+gbVrhoZltekOofO9w=";

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
