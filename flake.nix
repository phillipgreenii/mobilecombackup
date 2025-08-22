# Nix flake for mobilecombackup - Tool for processing mobile phone backup files
# This provides package-only distribution (development uses devbox exclusively)
{
  description = "Tool for processing mobile phone backup files";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        
        # Smart version detection using flake input detection (Option 1)
        # Provides near-perfect compatibility with existing build-version.sh
        detectVersion = let
          versionFile = builtins.readFile ./VERSION;
          baseVersion = pkgs.lib.removeSuffix "-dev" (pkgs.lib.removeSuffix "\n" versionFile);
          
          # Flake provides these attributes based on how it was fetched
          isTag = (self ? ref) && (pkgs.lib.hasPrefix "v" self.ref);
          tagVersion = if isTag then pkgs.lib.removePrefix "v" self.ref else null;
        in
          if isTag && tagVersion != null then
            # Clean version from tag (e.g., "2.0.0")
            tagVersion
          else if self ? rev then
            # Development version with git hash (e.g., "2.1.0-dev-g1234567")
            "${baseVersion}-dev-g${builtins.substring 0 7 self.rev}"
          else if self ? dirtyRev then
            # Local development with uncommitted changes
            "${baseVersion}-dev-dirty"
          else
            # Fallback when no git info available
            "${baseVersion}-dev";
            
      in {
        packages = {
          default = pkgs.buildGoModule rec {
            pname = "mobilecombackup";
            version = detectVersion;
            
            src = ./.;
            
            # Bootstrap with lib.fakeHash, then replace with real hash from build error
            #vendorHash = pkgs.lib.fakeHash;
            vendorHash = "sha256-R/1lfhkQN1Dr7qcusLVEmqv6s0dcNkHDyzHrQCEaSY8=";
            
            # Match current build flags from build-version.sh
            ldflags = [
              "-X main.Version=${version}"
              "-s -w"  # Strip debug info for smaller binary
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
          mobilecombackup = self.packages.${system}.default;
        };
        
        # Applications for nix run
        apps = {
          default = flake-utils.lib.mkApp {
            drv = self.packages.${system}.default;
          };
          mobilecombackup = self.apps.${system}.default;
        };
        
        # Quality checks for nix flake check
        checks = {
          # Verify package builds successfully
          build = self.packages.${system}.default;
          
          # Verify package can be installed and run
          integration = pkgs.runCommand "check-mobilecombackup-integration" {} ''
            # Install package
            ${self.packages.${system}.default}/bin/mobilecombackup --version
            
            # Verify version format
            VERSION_OUTPUT=$(${self.packages.${system}.default}/bin/mobilecombackup --version)
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
          help = pkgs.runCommand "check-mobilecombackup-help" {} ''
            # Test help command
            ${self.packages.${system}.default}/bin/mobilecombackup --help > help_output.txt
            
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
      }
    );
}
