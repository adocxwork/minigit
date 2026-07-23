package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"mgit/pkg/core"
)

var branchCmd = &cobra.Command{
	Use:   "branch [branch-name]",
	Short: "List or create branches",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoRoot, err := core.GetRepoPath()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Fatal:", err)
			os.Exit(1)
		}

		if len(args) == 0 {
			// List branches
			branches, err := core.ListBranches(repoRoot)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			currentBranch, _ := core.GetCurrentBranch(repoRoot)
			
			for _, b := range branches {
				if b == currentBranch {
					fmt.Printf("* %s\n", b)
				} else {
					fmt.Printf("  %s\n", b)
				}
			}
		} else {
			// Create branch
			branchName := args[0]
			headCommit, err := core.GetHEADCommit(repoRoot)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			if headCommit == "" {
				fmt.Fprintln(os.Stderr, "Error: not a valid object name: 'HEAD'")
				os.Exit(1)
			}

			if err := core.CreateBranch(repoRoot, branchName, headCommit); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(branchCmd)
}
