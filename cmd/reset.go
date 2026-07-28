package cmd

import (
	"fmt"
	"os"

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
	Run: func(cmd *cobra.Command, args []string) {
		repoRoot, err := core.GetRepoPath()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Fatal:", err)
			os.Exit(1)
		}

		target := args[0]
		mode := "mixed"
		if hard {
			mode = "hard"
		} else if soft {
			mode = "soft"
		} else if mixed {
			mode = "mixed"
		}

		if err := core.Reset(repoRoot, mode, target); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func init() {
	resetCmd.Flags().BoolVar(&hard, "hard", false, "Reset the working tree and index")
	resetCmd.Flags().BoolVar(&soft, "soft", false, "Do not touch the index file or the working tree at all")
	resetCmd.Flags().BoolVar(&mixed, "mixed", false, "Reset the index but not the working tree (default)")
	rootCmd.AddCommand(resetCmd)
}
