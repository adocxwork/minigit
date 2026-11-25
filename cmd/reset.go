package cmd

import (
	"github.com/spf13/cobra"
	"mgit/pkg/core"
)

var hard bool
var soft bool
var mixed bool

var resetCmd = &cobra.Command{
	Use:   "reset <commit-hash-or-branch>",
	Short: "Reset current HEAD to the specified state",
	Args:  cobra.ExactArgs(1),
	RunE: repoCommand(func(repoRoot string, args []string) error {
		mode := "mixed"
		if hard {
			mode = "hard"
		} else if soft {
			mode = "soft"
		}
		return core.Reset(repoRoot, mode, args[0])
	}),
}

func init() {
	resetCmd.Flags().BoolVar(&hard, "hard", false, "Reset the working tree and index")
	resetCmd.Flags().BoolVar(&soft, "soft", false, "Only move HEAD")
	resetCmd.Flags().BoolVar(&mixed, "mixed", false, "Reset the index (default)")
	resetCmd.MarkFlagsMutuallyExclusive("hard", "soft", "mixed")
	rootCmd.AddCommand(resetCmd)
}
