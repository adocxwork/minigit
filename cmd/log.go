package cmd

import (
	"github.com/spf13/cobra"
	"mgit/pkg/core"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show first-parent commit history",
	Args:  cobra.NoArgs,
	RunE: repoCommand(func(repoRoot string, args []string) error {
		return core.Log(repoRoot)
	}),
}

func init() {
	rootCmd.AddCommand(logCmd)
}
