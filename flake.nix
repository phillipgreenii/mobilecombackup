# Nix flake for mobilecombackup - Tool for processing mobile phone backup files
# This provides package-only distribution (development uses devbox exclusively)
{
  description = "Tool for processing mobile phone backup files";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
    # Without this follows, flake-parts pulls its own nixpkgs-lib snapshot and the
    # lock grows a second, unaligned view of the same library.
    flake-parts.inputs.nixpkgs-lib.follows = "nixpkgs";

    # Shared Nix infrastructure (pre-commit / treefmt / devshell / checks
    # flakeModules). Declared with follows so this flake's nixpkgs and
    # flake-parts are the single locked view (README "Cross-flake input
    # alignment"); without them flake.lock grows `_2`-suffixed duplicates.
    phillipgreenii-nix-base = {
      url = "github:phillipgreenii/nix-repo-base";
      inputs = {
        nixpkgs.follows = "nixpkgs";
        flake-parts.follows = "flake-parts";
      };
    };
  };

  outputs =
    inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      # Mirror flake-utils.lib.defaultSystems verbatim, as the sibling workspace
      # flakes do. This is the same system set the previous
      # flake-utils.lib.eachDefaultSystem wiring produced.
      systems = [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-darwin"
        "x86_64-linux"
      ];

      perSystem =
        { pkgs, config, ... }:
        let
          # Smart version detection using flake input detection (Option 1)
          # Provides near-perfect compatibility with existing build-version.sh
          detectVersion =
            let
              versionFile = builtins.readFile ./VERSION;
              baseVersion = pkgs.lib.removeSuffix "-dev" (pkgs.lib.removeSuffix "\n" versionFile);

              # Flake provides these attributes based on how it was fetched
              isTag = (inputs.self ? ref) && (pkgs.lib.hasPrefix "v" inputs.self.ref);
              tagVersion = if isTag then pkgs.lib.removePrefix "v" inputs.self.ref else null;
            in
            if isTag && tagVersion != null then
              # Clean version from tag (e.g., "2.0.0")
              tagVersion
            else if inputs.self ? rev then
              # Development version with git hash (e.g., "2.1.0-dev-g1234567")
              "${baseVersion}-dev-g${builtins.substring 0 7 inputs.self.rev}"
            else if inputs.self ? dirtyRev then
              # Local development with uncommitted changes
              "${baseVersion}-dev-dirty"
            else
              # Fallback when no git info available
              "${baseVersion}-dev";

          # flake-utils.lib.mkApp has no flake-parts equivalent; it expanded to
          # `{ type = "app"; program = "${drv}/bin/${drv.pname}"; }`, which is
          # written out by hand here.
          mainProgram = "${config.packages.default}/bin/mobilecombackup";
        in
        {
          packages = {
            default = pkgs.buildGoModule rec {
              pname = "mobilecombackup";
              version = detectVersion;

              src = ./.;

              # Bootstrap with lib.fakeHash, then replace with real hash from build error
              #vendorHash = pkgs.lib.fakeHash;
              vendorHash = "sha256-3+aJpFeRDFjC8a1f5JIgEFQE11H5pSjWyNqld6ObWPc=";

              # Match current build flags from build-version.sh
              ldflags = [
                "-X main.Version=${version}"
                "-s -w" # Strip debug info for smaller binary
              ];

              # Static binary matching current build
              env.CGO_ENABLED = 0;

              # Build from CLI entry point
              subPackages = [ "cmd/mobilecombackup" ];

              # Metadata for Nix package management
              meta = with pkgs.lib; {
                description = "Tool for processing mobile phone backup files";
                longDescription = ''
                  A command-line tool for processing mobile phone backup files including
                  call logs and SMS/MMS data in XML format. Provides deduplication,
                  attachment extraction, and organization by year.
                '';
                homepage = "https://github.com/phillipgreenii/mobilecombackup";
                license = licenses.mit;
                maintainers = [ ];
                platforms = platforms.unix;
                mainProgram = "mobilecombackup";
              };
            };

            # Alias for explicit access
            mobilecombackup = config.packages.default;
          };

          # Applications for nix run
          apps = {
            default = {
              type = "app";
              program = mainProgram;
            };
            mobilecombackup = {
              type = "app";
              program = mainProgram;
            };
          };

          # Quality checks for nix flake check
          checks = {
            # Verify package builds successfully
            build = config.packages.default;

            # Verify package can be installed and run
            integration = pkgs.runCommand "check-mobilecombackup-integration" { } ''
              # Install package
              ${config.packages.default}/bin/mobilecombackup --version

              # Verify version format
              VERSION_OUTPUT=$(${config.packages.default}/bin/mobilecombackup --version)
              echo "Version output: $VERSION_OUTPUT"

              # Check for proper version format (either clean version or dev version)
              if [[ "$VERSION_OUTPUT" =~ ^mobilecombackup\ version\ [0-9]+\.[0-9]+\.[0-9]+(-dev(-g[0-9a-f]{7}|-dirty)?)?$ ]]; then
                echo "✓ Version format correct"
                touch $out
              else
                echo "✗ Version format incorrect: $VERSION_OUTPUT"
                exit 1
              fi
            '';

            # Verify help command works
            help = pkgs.runCommand "check-mobilecombackup-help" { } ''
              # Test help command
              ${config.packages.default}/bin/mobilecombackup --help > help_output.txt

              # Verify help contains expected content
              if grep -q "processes call logs and SMS/MMS messages" help_output.txt; then
                echo "✓ Help output contains description"
                touch $out
              else
                echo "✗ Help output missing expected content"
                cat help_output.txt
                exit 1
              fi
            '';
          };
        };
    };
}
