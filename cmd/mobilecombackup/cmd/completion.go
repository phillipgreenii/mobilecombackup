package cmd

import (
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for mobilecombackup.

Shell completion provides tab-completion for commands, flags, and arguments,
making the CLI easier and faster to use.

To enable completions:

Bash:
  # Temporary (current session only):
  source <(mobilecombackup completion bash)
  
  # Permanent:
  echo 'source <(mobilecombackup completion bash)' >> ~/.bashrc
  
  # On some systems you may need to install bash-completion:
  # Ubuntu/Debian: apt install bash-completion
  # CentOS/RHEL: yum install bash-completion

Zsh:
  # Temporary (current session only):
  source <(mobilecombackup completion zsh)
  
  # Permanent:
  echo 'source <(mobilecombackup completion zsh)' >> ~/.zshrc
  
  # Note: You may need to add this to the beginning of ~/.zshrc:
  # autoload -U compinit && compinit

Fish:
  # Temporary (current session only):
  mobilecombackup completion fish | source
  
  # Permanent:
  mobilecombackup completion fish > ~/.config/fish/completions/mobilecombackup.fish

PowerShell:
  # Temporary (current session only):
  mobilecombackup completion powershell | Out-String | Invoke-Expression
  
  # Permanent: Add the above line to your PowerShell profile
  # To edit your profile: notepad $PROFILE

To verify completion is working:
  1. Start a new shell session (or source your profile)
  2. Type: mobilecombackup <TAB>
  3. You should see available commands and flags

Troubleshooting:
  - Bash: Ensure bash-completion is installed and sourced in ~/.bashrc
  - Zsh: Make sure compinit is loaded before the completion script
  - Fish: Check that ~/.config/fish/completions/ directory exists
  - All shells: Restart your shell or source the appropriate profile file

For more detailed shell-specific instructions, use:
  mobilecombackup completion [shell] --help`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
		case "zsh":
			return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
