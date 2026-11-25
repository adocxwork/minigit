package cmd

import (
	"github.com/spf13/cobra"
	"mgit/pkg/core"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the working tree status",
	Args:  cobra.NoArgs,
	RunE: repoCommand(func(repoRoot string, args []string) error {
		return core.Status(repoRoot)
	}),
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
