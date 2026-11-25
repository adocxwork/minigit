package cmd

import (
	"github.com/spf13/cobra"
	"mgit/pkg/core"
)

var addCmd = &cobra.Command{
	Use:   "add <path>",
	Short: "Stage file changes and tracked deletions",
	Args:  cobra.ExactArgs(1),
	RunE: repoCommand(func(repoRoot string, args []string) error {
		return core.Add(repoRoot, args[0])
	}),
}

func init() {
	rootCmd.AddCommand(addCmd)
}
