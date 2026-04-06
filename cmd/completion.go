package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

  $ source <(spawn-qdrant completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ spawn-qdrant completion bash > /etc/bash_completion.d/spawn-qdrant
  # macOS:
  $ spawn-qdrant completion bash > /usr/local/etc/bash_completion.d/spawn-qdrant

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ spawn-qdrant completion zsh > "${fpath[1]}/_spawn-qdrant"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ spawn-qdrant completion fish | source

  # To load completions for each session, execute once:
  $ spawn-qdrant completion fish > ~/.config/fish/completions/spawn-qdrant.fish

PowerShell:

  PS> spawn-qdrant completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> spawn-qdrant completion powershell > spawn-qdrant.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
