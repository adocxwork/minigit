package cmd

import (
	"github.com/spf13/cobra"
	"mgit/pkg/api"
	"mgit/pkg/core"
)

var port string

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the local web interface",
	Args:  cobra.NoArgs,
	RunE: func(*cobra.Command, []string) error {
		repoRoot, err := core.GetRepoPath()
		if err != nil {
			return err
		}
		return api.StartServer(repoRoot, port)
	},
}

func init() {
	uiCmd.Flags().StringVarP(&port, "port", "p", "8080", "Local UI port")
	rootCmd.AddCommand(uiCmd)
}
