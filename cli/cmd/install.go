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

	// Hide the .app from Spotlight and Finder by marking it as metadata-excluded.
	// This is the same mechanism Xcode uses for derived data folders.
	if err := hideFromSpotlight(appDir); err != nil {
		fmt.Printf("⚠ could not hide from Spotlight (non-fatal): %v\n", err)
	} else {
		fmt.Println("✔ hidden from Spotlight and Finder")
	}

	// Create symlink: Ghostify.app/Contents/MacOS/ghostify -> ghostify binary
	symlinkPath := filepath.Join(macOSDir, bundleExecutable)
	_ = os.Remove(symlinkPath) // remove stale symlink if present
	if err := os.Symlink(ghostifyBin, symlinkPath); err != nil {
		return fmt.Errorf("creating symlink: %w", err)
	}
	fmt.Printf("✔ symlinked %s -> %s\n", symlinkPath, ghostifyBin)

	// Register the bundle with Launch Services so it gets the bundle ID
	if err := registerBundle(appDir); err != nil {
		fmt.Printf("⚠ lsregister failed (non-fatal): %v\n", err)
	} else {
		fmt.Println("✔ registered bundle with Launch Services")
	}

	return nil
}

// hideFromSpotlight prevents the .app from appearing in Spotlight or Finder
// by writing a .metadata_never_index file and setting the xattr used by mdutil.
func hideFromSpotlight(appDir string) error {
	// 1. Write a .metadata_never_index sentinel file inside the bundle.
	//    Spotlight's mdworker respects this and skips the directory.
	sentinelPath := filepath.Join(appDir, ".metadata_never_index")
	if err := os.WriteFile(sentinelPath, []byte{}, 0o644); err != nil {
		return fmt.Errorf("writing .metadata_never_index: %w", err)
	}

	// 2. Set com.apple.metadata: com_apple_backup_excludeItem xattr so that
	//    the bundle is also excluded from Time Machine and Finder indexing.
	err := exec.Command(
		"xattr", "-w",
		"com.apple.metadata:com_apple_backup_excludeItem",
		"com.apple.backupd",
		appDir,
	).Run()
	if err != nil {
		// Non-fatal — the sentinel file alone is usually sufficient
		return fmt.Errorf("setting xattr: %w", err)
	}

	return nil
}

// resolveGhostifyBin finds the ghostify binary via `which ghostify`.
func resolveGhostifyBin() (string, error) {
	path, err := exec.LookPath(bundleExecutable)
	if err != nil {
		return "", errors.New("ghostify binary not found in PATH — is it installed via brew")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path, nil
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
