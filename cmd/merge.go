package cmd

import (
	"github.com/spf13/cobra"
	"mgit/pkg/core"
)

var mergeCmd = &cobra.Command{
	Use:   "merge <branch>",
	Short: "Merge another branch into the current history",
	Args:  cobra.ExactArgs(1),
	RunE: repoCommand(func(repoRoot string, args []string) error {
		return core.Merge(repoRoot, args[0])
	}),
}

func init() {
	rootCmd.AddCommand(mergeCmd)
}
