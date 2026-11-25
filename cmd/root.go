package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"mgit/pkg/core"
)

var rootCmd = &cobra.Command{
	Use:           "mgit",
	Short:         "An educational version control system",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func repoCommand(run func(repoRoot string, args []string) error) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, args []string) error {
		repoRoot, err := core.GetRepoPath()
		if err != nil {
			return err
		}
		unlock, err := core.LockRepository(repoRoot)
		if err != nil {
			return err
		}
		defer unlock()
		return run(repoRoot, args)
	}
}
