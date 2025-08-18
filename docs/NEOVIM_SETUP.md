# Neovim Setup for Go Development

This guide provides recommendations for setting up Neovim for Go development with this project.

## Essential Plugins

### LSP & Language Support
- **nvim-lspconfig** - LSP configuration for `gopls` (Go language server)
- **mason.nvim** + **mason-lspconfig.nvim** - Easy LSP server management
- **nvim-treesitter** - Better syntax highlighting and code parsing

### Completion & Snippets
- **nvim-cmp** - Autocompletion engine
- **cmp-nvim-lsp** - LSP completion source
- **LuaSnip** - Snippet engine
- **friendly-snippets** - Pre-built snippets including Go

### Go-Specific Tools
- **go.nvim** - Go development tools (test running, coverage, etc.)
- **gopher.nvim** - Go utilities (struct tags, impl generation)

### File Management
- **telescope.nvim** - Fuzzy finder for files, symbols, grep
- **nvim-tree.lua** or **neo-tree.nvim** - File explorer

### Git Integration
- **gitsigns.nvim** - Git status in gutter
- **fugitive.vim** - Git commands in Neovim

### Debugging
- **nvim-dap** - Debug Adapter Protocol
- **nvim-dap-go** - Go debugging configuration
- **nvim-dap-ui** - Debug UI

## Configuration Examples

### Lazy.nvim Plugin Manager Setup

Create `~/.config/nvim/lua/plugins/go.lua`:

```lua
return {
  -- LSP
  {
    "neovim/nvim-lspconfig",
    dependencies = {
      "williamboman/mason.nvim",
      "williamboman/mason-lspconfig.nvim",
    },
    config = function()
      require("mason").setup()
      require("mason-lspconfig").setup({
        ensure_installed = { "gopls" }
      })
      
      local lspconfig = require("lspconfig")
      lspconfig.gopls.setup({
        settings = {
          gopls = {
            analyses = {
              unusedparams = true,
            },
            staticcheck = true,
            gofumpt = true,
          },
        },
      })
    end,
  },
  
  -- Go tools
  {
    "ray-x/go.nvim",
    dependencies = {
      "ray-x/guihua.lua",
      "neovim/nvim-lspconfig",
      "nvim-treesitter/nvim-treesitter",
    },
    config = function()
      require("go").setup()
    end,
    event = {"CmdlineEnter"},
    ft = {"go", 'gomod'},
  }
}
```

### Key Mappings

Add to `~/.config/nvim/lua/config/keymaps.lua`:

```lua
-- Go-specific mappings
vim.keymap.set("n", "<leader>gr", ":GoRun<CR>", { desc = "Go Run" })
vim.keymap.set("n", "<leader>gt", ":GoTest<CR>", { desc = "Go Test" })
vim.keymap.set("n", "<leader>gc", ":GoCoverage<CR>", { desc = "Go Coverage" })
vim.keymap.set("n", "<leader>gf", ":GoFmt<CR>", { desc = "Go Format" })
vim.keymap.set("n", "<leader>gi", ":GoImport<CR>", { desc = "Go Import" })

-- Project-specific devbox commands
vim.keymap.set("n", "<leader>df", ":!devbox run formatter<CR>", { desc = "Run formatter" })
vim.keymap.set("n", "<leader>dt", ":!devbox run tests<CR>", { desc = "Run tests" })
vim.keymap.set("n", "<leader>dl", ":!devbox run linter<CR>", { desc = "Run linter" })
vim.keymap.set("n", "<leader>db", ":!devbox run build-cli<CR>", { desc = "Build CLI" })

-- LSP mappings (add these to your LSP configuration)
vim.keymap.set("n", "gd", vim.lsp.buf.definition, { desc = "Go to definition" })
vim.keymap.set("n", "gr", vim.lsp.buf.references, { desc = "Show references" })
vim.keymap.set("n", "K", vim.lsp.buf.hover, { desc = "Show hover documentation" })
vim.keymap.set("n", "<leader>rn", vim.lsp.buf.rename, { desc = "Rename symbol" })
vim.keymap.set("n", "<leader>ca", vim.lsp.buf.code_action, { desc = "Code actions" })
```

### Treesitter Configuration

Add to your Treesitter config:

```lua
require("nvim-treesitter.configs").setup({
  ensure_installed = { "go", "gomod", "gowork", "gosum" },
  highlight = {
    enable = true,
  },
  indent = {
    enable = true,
  },
})
```

## Development Workflow Integration

This setup integrates with the project's development workflow:

### Quality Commands
- `<leader>df` - Run devbox formatter
- `<leader>dt` - Run devbox tests  
- `<leader>dl` - Run devbox linter
- `<leader>db` - Build CLI

### Go Commands
- `<leader>gr` - Run current Go file
- `<leader>gt` - Run Go tests
- `<leader>gc` - Show test coverage
- `<leader>gf` - Format Go code
- `<leader>gi` - Add/organize imports

### LSP Features
- `gd` - Go to definition
- `gr` - Show references
- `K` - Show documentation
- `<leader>rn` - Rename symbol
- `<leader>ca` - Code actions

## Quick Start

1. **Install a plugin manager** like lazy.nvim
2. **Add the Go plugins** using the configuration above
3. **Install gopls**: Run `:MasonInstall gopls` in Neovim
4. **Configure LSP keymaps** for navigation and actions
5. **Set up go.nvim** for running tests and formatting
6. **Test the setup** by opening a Go file and using `gd` to jump to definitions

## NixOS and Home-Manager Setup

This section describes how to set up Neovim with NixOS and home-manager, keeping base configuration system-wide while loading Go-specific tooling only within this project via devbox.

### Base Neovim Configuration (Home-Manager)

Configure your base Neovim in `~/.config/home-manager/home.nix` or your home-manager modules:

```nix
{ config, pkgs, ... }:

{
  programs.neovim = {
    enable = true;
    defaultEditor = true;
    viAlias = true;
    vimAlias = true;
    
    # Base plugins for general development
    plugins = with pkgs.vimPlugins; [
      # Core functionality
      plenary-nvim
      
      # File management
      telescope-nvim
      telescope-fzf-native-nvim
      neo-tree-nvim
      
      # Git integration
      gitsigns-nvim
      vim-fugitive
      
      # UI enhancements
      lualine-nvim
      bufferline-nvim
      nvim-web-devicons
      
      # Treesitter for syntax highlighting
      nvim-treesitter.withAllGrammars
      
      # LSP base (without language-specific servers)
      nvim-lspconfig
      nvim-cmp
      cmp-nvim-lsp
      cmp-buffer
      cmp-path
      luasnip
      cmp_luasnip
      
      # General editing
      comment-nvim
      nvim-autopairs
      vim-surround
    ];
    
    extraConfig = ''
      lua << EOF
      -- Basic settings
      vim.opt.number = true
      vim.opt.relativenumber = true
      vim.opt.expandtab = true
      vim.opt.shiftwidth = 2
      vim.opt.tabstop = 2
      vim.opt.smartindent = true
      vim.opt.wrap = false
      vim.opt.termguicolors = true
      
      -- Set leader key
      vim.g.mapleader = " "
      
      -- Telescope setup
      require('telescope').setup{}
      
      -- Git signs
      require('gitsigns').setup{}
      
      -- Comment
      require('Comment').setup{}
      
      -- Autopairs
      require('nvim-autopairs').setup{}
      
      -- Basic keymaps
      vim.keymap.set('n', '<leader>ff', ':Telescope find_files<CR>')
      vim.keymap.set('n', '<leader>fg', ':Telescope live_grep<CR>')
      vim.keymap.set('n', '<leader>fb', ':Telescope buffers<CR>')
      vim.keymap.set('n', '<leader>e', ':Neotree toggle<CR>')
      EOF
    '';
  };
}
```

### Project-Specific Go Configuration

Create a project-local Neovim configuration that devbox will load:

#### 1. Update devbox.json

Add Neovim configuration to your `devbox.json`:

```json
{
  "packages": [
    "go@1.24",
    "gopls@latest",
    "golangci-lint@latest",
    // ... other packages
  ],
  "shell": {
    "init_hook": [
      // ... existing hooks
      "# Load project-specific Neovim config",
      "export NVIM_PROJECT_CONFIG=$PWD/.config/nvim",
      "if [ -d \"$NVIM_PROJECT_CONFIG\" ]; then",
      "  export XDG_CONFIG_HOME_NVIM=$PWD/.config",
      "  alias vim='nvim -u $NVIM_PROJECT_CONFIG/init.lua'",
      "  alias nvim='nvim -u $NVIM_PROJECT_CONFIG/init.lua'",
      "fi"
    ]
  }
}
```

#### 2. Create Project Neovim Configuration

Create `.config/nvim/init.lua` in your project root:

```lua
-- Load system Neovim config first
local system_config = vim.fn.expand("~/.config/nvim/init.lua")
if vim.fn.filereadable(system_config) == 1 then
  vim.cmd("source " .. system_config)
end

-- Project-specific Go configuration
local project_root = vim.fn.getcwd()

-- Ensure gopls is available (provided by devbox)
local lspconfig = require('lspconfig')

-- Configure gopls for Go development
lspconfig.gopls.setup({
  cmd = {'gopls'},
  filetypes = {'go', 'gomod', 'gowork', 'gotmpl'},
  root_dir = lspconfig.util.root_pattern('go.mod', '.git'),
  settings = {
    gopls = {
      analyses = {
        unusedparams = true,
        shadow = true,
      },
      staticcheck = true,
      gofumpt = true,
      usePlaceholders = true,
      completeUnimported = true,
    },
  },
  on_attach = function(client, bufnr)
    -- Go-specific keymaps
    local opts = { buffer = bufnr, noremap = true, silent = true }
    vim.keymap.set('n', 'gd', vim.lsp.buf.definition, opts)
    vim.keymap.set('n', 'gr', vim.lsp.buf.references, opts)
    vim.keymap.set('n', 'gi', vim.lsp.buf.implementation, opts)
    vim.keymap.set('n', 'K', vim.lsp.buf.hover, opts)
    vim.keymap.set('n', '<leader>rn', vim.lsp.buf.rename, opts)
    vim.keymap.set('n', '<leader>ca', vim.lsp.buf.code_action, opts)
    vim.keymap.set('n', '<leader>f', function()
      vim.lsp.buf.format({ async = true })
    end, opts)
  end,
})

-- Project-specific keymaps for devbox commands
vim.keymap.set('n', '<leader>df', ':!devbox run formatter<CR>', { desc = 'Run formatter' })
vim.keymap.set('n', '<leader>dt', ':!devbox run test-unit<CR>', { desc = 'Run unit tests' })
vim.keymap.set('n', '<leader>dT', ':!devbox run tests<CR>', { desc = 'Run all tests' })
vim.keymap.set('n', '<leader>dl', ':!devbox run linter<CR>', { desc = 'Run linter' })
vim.keymap.set('n', '<leader>db', ':!devbox run build-cli<CR>', { desc = 'Build CLI' })
vim.keymap.set('n', '<leader>dc', ':!devbox run coverage-summary<CR>', { desc = 'Coverage summary' })

-- Code analysis tools
vim.keymap.set('n', '<leader>af', ':!ast-grep --pattern "func $NAME($$$) $RET { $$$ }"<CR>', { desc = 'Find function definitions' })
vim.keymap.set('n', '<leader>ae', ':!ast-grep --pattern "if err != nil { $$$ }"<CR>', { desc = 'Find error handling' })
vim.keymap.set('n', '<leader>at', ':!ast-grep --pattern "func Test$_($$$) { $$$ }"<CR>', { desc = 'Find test functions' })

-- Go-specific commands
vim.api.nvim_create_user_command('GoTest', '!go test -v ./%:h', {})
vim.api.nvim_create_user_command('GoTestFunc', '!go test -v -run ' .. vim.fn.expand('<cword>') .. ' ./%:h', {})
vim.api.nvim_create_user_command('GoRun', '!go run %', {})
vim.api.nvim_create_user_command('GoBuild', '!go build %', {})

-- Auto-format Go files on save
vim.api.nvim_create_autocmd('BufWritePre', {
  pattern = '*.go',
  callback = function()
    vim.lsp.buf.format({ async = false })
  end,
})

-- Configure completion for Go
local cmp = require('cmp')
cmp.setup.filetype('go', {
  sources = cmp.config.sources({
    { name = 'nvim_lsp' },
    { name = 'luasnip' },
  }, {
    { name = 'buffer' },
    { name = 'path' },
  })
})

print("✅ Loaded project-specific Go configuration")
```

#### 3. Alternative: Using direnv with devbox

If you prefer using direnv, create `.envrc` in your project:

```bash
# .envrc
use devbox

# Set up project-specific Neovim config
export NVIM_APPNAME="nvim-go"
export XDG_CONFIG_HOME="$PWD/.config"

# Create wrapper function for vim/nvim
vim() {
  if [ -f "$PWD/.config/nvim/init.lua" ]; then
    nvim -u "$PWD/.config/nvim/init.lua" "$@"
  else
    nvim "$@"
  fi
}
```

### Minimal Go Plugin Setup (Optional)

If you want to use plugins specifically for Go without installing them system-wide, create `.config/nvim/lua/go-plugins.lua`:

```lua
-- Lazy-load Go plugins only when editing Go files
vim.api.nvim_create_autocmd({'BufEnter', 'BufWinEnter'}, {
  pattern = {'*.go', '*.mod', '*.sum'},
  callback = function()
    -- Load go.nvim functionality
    vim.keymap.set('n', '<leader>gr', ':GoRun<CR>', { buffer = true })
    vim.keymap.set('n', '<leader>gt', ':GoTest<CR>', { buffer = true })
    vim.keymap.set('n', '<leader>gc', ':GoCoverage<CR>', { buffer = true })
    vim.keymap.set('n', '<leader>gi', ':GoImport<CR>', { buffer = true })
    vim.keymap.set('n', '<leader>gf', ':GoFmt<CR>', { buffer = true })
  end
})
```

### Testing the Setup

1. **Outside the project**: Run `vim` to get your base Neovim with file management and Git support
2. **Inside the project**: 
   ```bash
   cd /path/to/mobilecombackup
   devbox shell
   vim  # Now includes Go LSP and project commands
   ```

3. **Verify Go support is loaded**:
   - Open a `.go` file
   - Check `:LspInfo` shows gopls running
   - Test `<leader>dt` runs tests
   - Test `gd` jumps to definition

### Benefits of This Approach

- **Separation of concerns**: System-wide config stays minimal and general
- **Project isolation**: Go tools only load when working on Go projects
- **Reproducible**: Team members get the same Go setup via devbox
- **Flexible**: Can extend for other language projects similarly
- **Clean**: No Go cruft in your system when working on non-Go projects

## Additional Tips

- Use `:GoInstallBinaries` to install additional Go tools
- Configure auto-formatting on save with `gopls`
- Set up debugging with nvim-dap for stepping through code
- Use Telescope for fuzzy finding across your codebase
- Consider adding vim-test for more flexible test running

This configuration provides a complete Go development environment that integrates well with this project's devbox-based workflow.