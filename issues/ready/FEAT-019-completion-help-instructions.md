# FEAT-019: Include shell completion setup instructions in help

## Status
- **Completed**: 
- **Priority**: low

## Overview
Add instructions in the completion command help text on how to apply the generated completion scripts to the user's shell, making it easier for users to enable tab completion.

## Background
The current completion command generates shell completion scripts but doesn't explain how to use them. Users need clear instructions on how to apply these scripts to enable tab completion in their shell.

## Requirements
### Functional Requirements
- [ ] Add setup instructions to the completion command help text
- [ ] Include instructions for bash, zsh, fish, and powershell
- [ ] Show both temporary (current session) and permanent setup methods
- [ ] Include OS-specific notes where relevant (macOS, Linux, Windows)

### Non-Functional Requirements
- [ ] Instructions must be clear and easy to follow
- [ ] Commands should be copy-pasteable
- [ ] Include troubleshooting tips for common issues

## Design
### Approach
Enhance the completion command's help text with detailed setup instructions for each supported shell.

### Implementation Notes
Update the completion command description and long help text to include:
1. Brief explanation of what shell completion provides
2. Per-shell setup instructions with examples
3. How to verify completion is working
4. Common troubleshooting steps

Example help output:
```
Generate shell completion scripts

Usage:
  mobilecombackup completion [bash|zsh|fish|powershell]

To enable completions:

Bash:
  # Temporary (current session only):
  source <(mobilecombackup completion bash)
  
  # Permanent:
  echo 'source <(mobilecombackup completion bash)' >> ~/.bashrc

Zsh:
  # Temporary (current session only):
  source <(mobilecombackup completion zsh)
  
  # Permanent:
  echo 'source <(mobilecombackup completion zsh)' >> ~/.zshrc
  
  # Note: You may need to add this to the beginning of ~/.zshrc:
  autoload -U compinit && compinit

Fish:
  mobilecombackup completion fish | source
  
  # Permanent:
  mobilecombackup completion fish > ~/.config/fish/completions/mobilecombackup.fish

PowerShell:
  mobilecombackup completion powershell | Out-String | Invoke-Expression
  
  # Permanent: Add the above line to your PowerShell profile
  # Run: notepad $PROFILE
```

## Tasks
- [ ] Update completion command help text
- [ ] Add setup instructions for each shell
- [ ] Add troubleshooting section
- [ ] Test instructions on each platform/shell
- [ ] Update documentation

## Testing
### Unit Tests
- Verify help text is displayed correctly

### Integration Tests
- Test completion setup on each supported shell
- Verify completions work after following instructions

### Edge Cases
- Non-standard shell configurations
- Permission issues with profile files
- Missing shell completion dependencies

## Risks and Mitigations
- **Risk**: Instructions may not work for all shell configurations
  - **Mitigation**: Include troubleshooting section and links to shell-specific docs

## References
- Cobra completion documentation: https://github.com/spf13/cobra/blob/main/site/content/completions/_index.md
- Shell-specific completion guides

## Notes
This is a documentation enhancement that improves user experience without changing functionality.