package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Refresh the Ghostify.app bundle and symlink after a brew upgrade",
	RunE:  runUpdate,
}

func runUpdate(_ *cobra.Command, _ []string) error {
	appDir := expandHome(appBundlePath)

	// Remove old symlink only — keep the bundle structure so we don't lose the plist
	symlinkPath := filepath.Join(appDir, "Contents", "MacOS", bundleExecutable)
	if err := os.Remove(symlinkPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing old symlink: %w", err)
	}
	fmt.Println("✔ removed old symlink")

	return install()
}
