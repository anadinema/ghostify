package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove Ghostify.app bundle (called automatically by brew uninstall)",
	RunE:  runUninstall,
}

func runUninstall(_ *cobra.Command, _ []string) error {
	appDir := expandHome(appBundlePath)

	if err := os.RemoveAll(appDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing Ghostify.app: %w", err)
	}
	fmt.Println("✔ removed ~/Applications/Ghostify.app")
	return nil
}
