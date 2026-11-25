package cmd

import (
	"github.com/spf13/cobra"
	"mgit/pkg/core"
)

var checkoutCmd = &cobra.Command{
	Use:   "checkout <branch|commit>",
	Short: "Switch to a branch or commit",
	Args:  cobra.ExactArgs(1),
	RunE: repoCommand(func(repoRoot string, args []string) error {
		return core.Checkout(repoRoot, args[0])
	}),
}

func init() {
	rootCmd.AddCommand(checkoutCmd)
}
