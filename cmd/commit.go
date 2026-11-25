package cmd

import (
	"github.com/spf13/cobra"
	"mgit/pkg/core"
)

var message string

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Record staged changes to the repository",
	Args:  cobra.NoArgs,
	RunE: repoCommand(func(repoRoot string, _ []string) error {
		return core.CreateCommit(repoRoot, message, nil)
	}),
}

func init() {
	commitCmd.Flags().StringVarP(&message, "message", "m", "", "Commit message")
	commitCmd.MarkFlagRequired("message")
	rootCmd.AddCommand(commitCmd)
}
