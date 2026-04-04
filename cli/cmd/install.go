package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	appBundlePath    = "~/Applications/Ghostify.app"
	bundleExecutable = "ghostify"
	bundleID         = "dev.anadinema.wrapper.ghostify"

	infoPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleIdentifier</key>
    <string>dev.anadinema.wrapper.ghostify</string>
    <key>CFBundleName</key>
    <string>Ghostify</string>
    <key>CFBundleExecutable</key>
    <string>ghostify</string>
    <key>CFBundleVersion</key>
    <string>1.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>LSUIElement</key>
    <false/>
</dict>
</plist>
`
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Link the ghostify binary and create the Ghostify.app bundle",
	RunE:  runInstall,
}

func runInstall(_ *cobra.Command, _ []string) error {
	return install()
}

// install is called by both installCmd and setupCmd.
func install() error {
	ghostifyBin, err := resolveGhostifyBin()
	if err != nil {
		return err
	}

	appDir := expandHome(appBundlePath)
	macOSDir := filepath.Join(appDir, "Contents", "MacOS")
	contentsDir := filepath.Join(appDir, "Contents")

	// Create bundle directory structure
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		return fmt.Errorf("creating app bundle directories: %w", err)
	}

	// Write Info.plist
	plistPath := filepath.Join(contentsDir, "Info.plist")
	if err := os.WriteFile(plistPath, []byte(infoPlist), 0o644); err != nil {
		return fmt.Errorf("writing Info.plist: %w", err)
	}
	fmt.Println("✔ wrote Info.plist")

	// Create symlink: Ghostify.app/Contents/MacOS/ghostify -> ghostify binary
	symlinkPath := filepath.Join(macOSDir, bundleExecutable)
	_ = os.Remove(symlinkPath) // remove stale symlink if present
	if err := os.Symlink(ghostifyBin, symlinkPath); err != nil {
		return fmt.Errorf("creating symlink: %w", err)
	}
	fmt.Printf("✔ symlinked %s -> %s\n", symlinkPath, ghostifyBin)

	// Register the bundle with Launch Services
	if err := registerBundle(appDir); err != nil {
		// Non-fatal — AltTab may still pick it up
		fmt.Printf("⚠ lsregister failed (non-fatal): %v\n", err)
	} else {
		fmt.Println("✔ registered bundle with Launch Services")
	}

	return nil
}

// resolveGhostifyBin finds the ghostify binary via `which ghostify`.
func resolveGhostifyBin() (string, error) {
	path, err := exec.LookPath(bundleExecutable)
	if err != nil {
		return "", errors.New("ghostify binary not found in PATH — is it installed via brew")
	}
	// Resolve symlinks so we point to the real binary
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path, nil // fall back to unresolved
	}
	return resolved, nil
}

func registerBundle(appDir string) error {
	lsregister := "/System/Library/Frameworks/CoreServices.framework/Versions/A/Frameworks/LaunchServices.framework/Versions/A/Support/lsregister"
	return exec.Command(lsregister, "-f", appDir).Run()
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
