package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ghost-spot",
	Short: "Ghostify — spotify_player with macOS integration",
	Long:  `ghost-spot manages the Ghostify installation: app bundle, binary linking, and spotify_player config.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(uninstallCmd)
}
