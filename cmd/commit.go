package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"mgit/pkg/core"
)

var message string

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Record changes to the repository",
	Run: func(cmd *cobra.Command, args []string) {
		repoRoot, err := core.GetRepoPath()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Fatal:", err)
			os.Exit(1)
		}

		if message == "" {
			fmt.Fprintln(os.Stderr, "Error: commit message is required")
			os.Exit(1)
		}

		if err := core.CreateCommit(repoRoot, message); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func init() {
	commitCmd.Flags().StringVarP(&message, "message", "m", "", "Commit message")
	commitCmd.MarkFlagRequired("message")
	rootCmd.AddCommand(commitCmd)
}
