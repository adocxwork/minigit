package cmd

import (
	"github.com/spf13/cobra"
	"mgit/pkg/core"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new mgit repository",
	Args:  cobra.NoArgs,
	RunE: func(*cobra.Command, []string) error {
		return core.InitRepo()
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
