package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"mgit/pkg/api"
	"mgit/pkg/core"
)

var port string

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the web-based graphical user interface",
	Run: func(cmd *cobra.Command, args []string) {
		repoRoot, err := core.GetRepoPath()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Fatal:", err)
			os.Exit(1)
		}

		if err := api.StartServer(repoRoot, port); err != nil {
			fmt.Fprintln(os.Stderr, "Error starting server:", err)
			os.Exit(1)
		}
	},
}

func init() {
	uiCmd.Flags().StringVarP(&port, "port", "p", "8080", "Port to run the UI server on")
	rootCmd.AddCommand(uiCmd)
}
