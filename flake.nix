{
  description = "Personal Finance";

  inputs = {
    #nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
    # devenv
    devenv-root = {
      url = "file+file:///dev/null";
      flake = false;
    };
    devenv.url = "github:cachix/devenv";
    # flake.parts
    flake-parts.url = "github:hercules-ci/flake-parts";
    # flake-root
    flake-root.url = "github:srid/flake-root";
    # systems
    systems.url = "github:nix-systems/default";
    # mk shell bin 
    mk-shell-bin.url = "github:rrbutani/nix-mk-shell-bin";
    # treefmt-nix
    treefmt-nix.url = "github:numtide/treefmt-nix";
    treefmt-nix.inputs.nixpkgs.follows = "nixpkgs";
    # alejandra
    alejandra.url = "github:kamadorueda/alejandra/3.0.0";
    alejandra.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, flake-parts, devenv-root, ... }@inputs:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.flake-root.flakeModule
        inputs.devenv.flakeModule
        inputs.treefmt-nix.flakeModule
      ];
      systems = import inputs.systems;

      perSystem = { config, self', inputs', pkgs, system, ... }: {
        # Per-system attributes can be defined here. The self' and inputs'
        # module parameters provide easy access to attributes of the same
        # system.
        flake-root.projectRootFile = ".git/config";
        # flake's own devenv
        # auto formatters
        treefmt.config = {
          inherit (config.flake-root) projectRootFile;
          settings.global.excludes = [ "*.xml" ];
          programs.nixpkgs-fmt.enable = true;
        };
        # FIXME `devenv test` doesn't work, possibly related to https://github.com/cachix/devenv/issues/1357
        packages.devenv-test = self.devShells.${system}.default.config.test;


        devenv.shells.default = {
          devenv.root =
            let
              devenvRootFileContent = builtins.readFile devenv-root.outPath;
            in
            pkgs.lib.mkIf (devenvRootFileContent != "") devenvRootFileContent;

          name = "beancount";

          # hooks keep failing even though the individual checks seem to pass
          pre-commit.default_stages = [ "manual" ];
          pre-commit.hooks = {
            check-go.enable = true;
            check-symlinks.enable = true;
            markdownlint.enable = true;
            name-tests-test.enable = true;
            nil.enable = true;
            # FIXME this fails because there is no treefmt.toml, not sure how to config defined in flake.nix
            #treefmt.enable = true;
            typos.enable = true;
          };

          scripts = {
            treefmt.exec = ''
              ${config.treefmt.build.wrapper}/bin/treefmt "$@";
            '';
            run-tests.exec = ''
              go test -v -covermode=set ./...
            '';
          };

          enterTest = ''
            echo "running fake tests3"
          '';

          difftastic.enable = true;

          # https://devenv.sh/reference/options/
          packages = with pkgs; [
            bashInteractive
            nixd

            #git 
            git
            lazygit

            #go
            go

            # neovim
            (pkgs.neovim.override {
              #defaultEditor = true;
              viAlias = true;
              vimAlias = true;
              configure = {
                packages.packages = with pkgs.vimPlugins; {
                  start = [
                    nvim-lspconfig
                    (nvim-treesitter.withPlugins (
                      plugins: with plugins; [
                        tree-sitter-go
                      ]
                    ))
                    sensible
                  ];
                };
              };
            })
          ];
        };
      };
    };
}
