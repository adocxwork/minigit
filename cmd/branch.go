package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"mgit/pkg/core"
)

var branchCmd = &cobra.Command{
	Use:   "branch [branch-name]",
	Short: "List or create branches",
	Args:  cobra.MaximumNArgs(1),
	RunE: repoCommand(func(repoRoot string, args []string) error {
		if len(args) > 0 {
			head, err := core.GetHEADCommit(repoRoot)
			if err != nil {
				return err
			}
			return core.CreateBranch(repoRoot, args[0], head)
		}
		branches, err := core.ListBranches(repoRoot)
		if err != nil {
			return err
		}
		current, err := core.GetCurrentBranch(repoRoot)
		if err != nil {
			return err
		}
		for _, branch := range branches {
			if branch == current {
				fmt.Printf("* %s\n", branch)
			} else {
				fmt.Printf("  %s\n", branch)
			}
		}
		return nil
	}),
}

func init() {
	rootCmd.AddCommand(branchCmd)
}
