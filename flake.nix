# Nix flake for mobilecombackup - Tool for processing mobile phone backup files
# This provides package-only distribution (development uses devbox exclusively)
{
  description = "Tool for processing mobile phone backup files";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-26.05-darwin";
    flake-parts.url = "github:hercules-ci/flake-parts";
    # Without this follows, flake-parts pulls its own nixpkgs-lib snapshot and the
    # lock grows a second, unaligned view of the same library.
    flake-parts.inputs.nixpkgs-lib.follows = "nixpkgs";

    # Shared Nix infrastructure (pre-commit / treefmt / devshell / checks
    # flakeModules, the Go builder factory `lib.mkGoBuilders`, and the
    # re-exported gomod2nix overlay). Declared with follows so this flake's
    # nixpkgs and flake-parts are the single locked view (README "Cross-flake
    # input alignment"); without them flake.lock grows `_2`-suffixed duplicates.
    phillipgreenii-nix-base = {
      url = "github:phillipgreenii/nix-repo-base";
      inputs = {
        nixpkgs.follows = "nixpkgs";
        flake-parts.follows = "flake-parts";
      };
    };
  };

  outputs =
    inputs@{ flake-parts, phillipgreenii-nix-base, ... }:
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
        { pkgs, system, config, ... }:
        let
          # ADR 0006 / bead tc-5lxy.1 (Option A): the human-facing base version is
          # read from the committed VERSION file and NOTHING else. mkGoApp appends
          # an 8-hex digest of this package's own source tree, so the built
          # `--version` reads `2.0.0-<8hex>`.
          #
          # Deriving this from `self.rev` / `self.ref` / `self.dirtyRev` (as the
          # pre-mkGoBinary `detectVersion` block did) is FORBIDDEN: it puts the
          # repo git rev back into the derivation's `version`, so the drvPath
          # changes on every commit and the package rebuilds for edits it does not
          # contain — exactly the per-commit churn ADR 0006 exists to remove. See
          # the CONSUMER CONSTRAINT note on `baseVersion` in nix-repo-base's
          # lib/go-builders.nix.
          #
          # The `-dev` suffix is stripped so the value is a bare semver: the digest
          # is what marks a build as non-release, and `checks.integration` asserts
          # the `<semver>-<8hex>` shape.
          baseVersion = pkgs.lib.removeSuffix "-dev" (
            pkgs.lib.removeSuffix "\n" (builtins.readFile ./VERSION)
          );

          goBuilders = phillipgreenii-nix-base.lib.mkGoBuilders {
            inherit pkgs;
            inherit (pkgs) lib;
            self = inputs.self;
          };

          # flake-utils.lib.mkApp has no flake-parts equivalent; it expanded to
          # `{ type = "app"; program = "${drv}/bin/${drv.pname}"; }`, which is
          # written out by hand here.
          mainProgram = "${config.packages.default}/bin/mobilecombackup";
        in
        {
          # gomod2nix's overlay supplies pkgs.buildGoApplication, which mkGoBinary
          # (via mkGoApp) requires. Sourced from nix-base (overlays.gomod2nix) so
          # this flake needs no direct gomod2nix input. Mirrors bb / nix-repo-base.
          _module.args.pkgs = import inputs.nixpkgs {
            inherit system;
            overlays = [ phillipgreenii-nix-base.overlays.gomod2nix ];
          };

          packages = {
            # Built via mkGoBinary (ADR 0008): dependencies come from the committed
            # gomod2nix.toml as per-module content-addressed FODs, so there is no
            # `vendorHash` to hand-maintain. On top of the build it generates a man
            # page (help2man) and bash/zsh/fish completions from cobra's
            # `mobilecombackup completion <shell>`, plus (via extraPostInstall) the
            # hand-authored tldr page.
            #
            # mkGoBinary's arg set is CLOSED: `name` (not `pname`) and a top-level
            # `description` (help2man's `--name` tagline and meta.description).
            default = goBuilders.mkGoBinary {
              name = "mobilecombackup";
              src = pkgs.lib.cleanSource ./.;
              description = "Tool for processing mobile phone backup files";

              inherit baseVersion;

              # cmd/mobilecombackup is not the only `package main` in this module
              # (demos/dashboard/main.go is another) and mkGoBinary defaults
              # subPackages to null = build every main. Pin the entrypoint so
              # $out/bin holds exactly one file.
              subPackages = [ "cmd/mobilecombackup" ];

              gomod2nixToml = ./gomod2nix.toml;

              # Match the historical build flags. mkGoApp appends its own
              # `-X main.Version=<version>` after these, so version injection is
              # never lost. cmd/mobilecombackup/main.go declares `var Version` in
              # package main, which is mkGoBinary's default versionPath.
              ldflags = [ "-s -w" ]; # Strip debug info for smaller binary
              env.CGO_ENABLED = 0; # Static binary matching the historical build

              # tldr page: not generated by mkGoBinary, so install the
              # hand-authored source into the workspace tldr convention path
              # ($out/share/tldr/pages.common/<name>.md).
              extraPostInstall = ''
                mkdir -p $out/share/tldr/pages.common
                cp ${./docs/tldr/mobilecombackup.md} $out/share/tldr/pages.common/mobilecombackup.md
              '';

              # Merged OVER mkGoBinary's own { description, mainProgram }, so the
              # metadata the pre-mkGoBinary buildGoModule call carried is retained.
              meta = with pkgs.lib; {
                longDescription = ''
                  A command-line tool for processing mobile phone backup files including
                  call logs and SMS/MMS data in XML format. Provides deduplication,
                  attachment extraction, and organization by year.
                '';
                homepage = "https://github.com/phillipgreenii/mobilecombackup";
                license = licenses.mit;
                maintainers = [ ];
                platforms = platforms.unix;
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

              # ADR 0006 digest versioning (bead tc-5lxy.1, Option A): mkGoApp
              # derives the version as the baseVersion followed by an 8-hex digest
              # of this package's own source, so the string is always
              # `<semver>-<8hex>`. The semver half is deliberately NOT pinned to
              # 0.0.0 — it comes from the VERSION file via baseVersion, so a later
              # VERSION bump does not re-break this check.
              if [[ "$VERSION_OUTPUT" =~ ^mobilecombackup\ version\ [0-9]+\.[0-9]+\.[0-9]+-[0-9a-f]{8}$ ]]; then
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

            # Durable regression guard for everything mkGoBinary layers on top of
            # the plain Go build. help2man and each `completion <shell>` call are
            # wrapped in mkGoBinary's `_try` helper, which `rm -f`s the artifact and
            # only WARNS when generation fails — so ABSENCE is the silent failure
            # mode and only an explicit check catches it.
            packaging = pkgs.runCommand "check-mobilecombackup-packaging" { } ''
              pkg=${config.packages.default}
              fail=0
              check_file() {
                if [ -s "$1" ]; then
                  echo "✓ $2: $1"
                else
                  echo "✗ $2 missing or empty: $1"
                  fail=1
                fi
              }

              # stdenv's compressManPages fixup hook gzips the page, so the
              # installed name is mobilecombackup.1.gz; accept either spelling so
              # this guard does not depend on that hook staying enabled.
              if [ -s "$pkg/share/man/man1/mobilecombackup.1" ] || [ -s "$pkg/share/man/man1/mobilecombackup.1.gz" ]; then
                echo "✓ man page: $(ls "$pkg"/share/man/man1/mobilecombackup.1*)"
              else
                echo "✗ man page missing or empty: $pkg/share/man/man1/mobilecombackup.1[.gz]"
                fail=1
              fi

              check_file "$pkg/share/bash-completion/completions/mobilecombackup" "bash completion"
              check_file "$pkg/share/zsh/site-functions/_mobilecombackup" "zsh completion"
              check_file "$pkg/share/fish/vendor_completions.d/mobilecombackup.fish" "fish completion"
              check_file "$pkg/share/tldr/pages.common/mobilecombackup.md" "tldr page"

              # subPackages pins the entrypoint; demos/dashboard must NOT be built.
              bins=$(ls "$pkg/bin")
              echo "Contents of \$out/bin: $bins"
              if [ "$bins" = "mobilecombackup" ]; then
                echo "✓ \$out/bin holds exactly the mobilecombackup binary"
              else
                echo "✗ \$out/bin should hold exactly 'mobilecombackup'"
                fail=1
              fi

              if [ "$fail" -ne 0 ]; then
                exit 1
              fi
              touch $out
            '';

            # golangci-lint (offline, gomod2nix vendor env) via mkGoLint --
            # the Tier-1 lint gate (tc-5lxy.19). `config` lives outside
            # `src`, so it must be passed explicitly, or golangci-lint falls
            # back to its defaults and loses this repo's .golangci.yml
            # settings (in particular the goconst disable and the
            # max-issues-per-linter/max-same-issues=0 overrides — see that
            # file's own comments for why both matter here).
            lint = goBuilders.mkGoLint {
              pname = "mobilecombackup-golangci";
              src = pkgs.lib.cleanSource ./.;
              gomod2nixToml = ./gomod2nix.toml;
              config = ./.golangci.yml;
            };
          };
        };
    };
}
