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

