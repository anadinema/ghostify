package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
)

var (
	flagPreset     string
	flagDeviceName string
	flagNoCompName bool
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install Ghostify and write spotify-player config",
	Long: `setup runs install and then writes ~/.config/spotify-player/app.toml.

Device name resolution:
  --preset work               →  "<ComputerName> Work - Ghostify"
  --preset personal           →  "<ComputerName> - Ghostify"
  --preset work --no-comp-name  →  "Work MacBook Ghostify"
  --preset personal --no-comp-name  →  "MacBook Ghostify"
  --device-name "My Device"   →  "My Device"  (--no-comp-name is ignored)`,
	RunE: runSetup,
}

func init() {
	setupCmd.Flags().StringVar(&flagPreset, "preset", "", "Device name preset: work or personal")
	setupCmd.Flags().StringVar(&flagDeviceName, "device-name", "", "Explicit device name (overrides --preset)")
	setupCmd.Flags().BoolVar(&flagNoCompName, "no-comp-name", false, "Use generic device name instead of macOS ComputerName (only with --preset)")
}

func runSetup(_ *cobra.Command, _ []string) error {
	// Validate flag combinations
	if flagNoCompName && flagPreset == "" {
		return errors.New("--no-comp-name requires --preset")
	}
	if flagPreset != "" && flagPreset != "work" && flagPreset != "personal" {
		return errors.New("--preset must be 'work' or 'personal'")
	}
	if flagPreset == "" && flagDeviceName == "" {
		return errors.New("one of --preset or --device-name is required")
	}

	deviceName, err := resolveDeviceName()
	if err != nil {
		return err
	}

	// Run install first
	if err := install(); err != nil {
		return err
	}

	// Write config
	if err := writeAppToml(deviceName); err != nil {
		return err
	}

	fmt.Printf("✔ setup complete — device name: %q\n", deviceName)
	return nil
}

func resolveDeviceName() (string, error) {
	// --device-name wins unconditionally
	if flagDeviceName != "" {
		return flagDeviceName, nil
	}

	if flagNoCompName {
		switch flagPreset {
		case "work":
			return "Work MacBook Ghostify", nil
		case "personal":
			return "MacBook Ghostify", nil
		}
	}

	compName, err := getComputerName()
	if err != nil {
		return "", fmt.Errorf("could not get macOS ComputerName (try --no-comp-name): %w", err)
	}

	switch flagPreset {
	case "work":
		return compName + " Work - Ghostify", nil
	case "personal":
		return compName + " - Ghostify", nil
	}

	return "", errors.New("unreachable")
}

func getComputerName() (string, error) {
	out, err := exec.Command("scutil", "--get", "ComputerName").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// writeAppToml writes the full app.toml, only varying the device name fields.
func writeAppToml(deviceName string) error {
	configDir := expandHome("~/.config/spotify-player")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	configPath := filepath.Join(configDir, "app.toml")

	// If a config already exists, update only the device name fields in place.
	if _, err := os.Stat(configPath); err == nil {
		return updateDeviceNameInConfig(configPath, deviceName)
	}

	// Otherwise write the full default config.
	cfg := defaultConfig(deviceName)
	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("creating app.toml: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return fmt.Errorf("encoding app.toml: %w", err)
	}
	fmt.Printf("✔ wrote %s\n", configPath)
	return nil
}

// updateDeviceNameInConfig reads the existing TOML, patches the two device
// name fields, and writes it back.
func updateDeviceNameInConfig(path, deviceName string) error {
	var cfg map[string]interface{}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return fmt.Errorf("reading existing app.toml: %w", err)
	}

	cfg["default_device"] = deviceName

	if device, ok := cfg["device"].(map[string]interface{}); ok {
		device["name"] = deviceName
	} else {
		cfg["device"] = map[string]interface{}{"name": deviceName}
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("writing app.toml: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return fmt.Errorf("encoding app.toml: %w", err)
	}
	fmt.Printf("✔ updated device name in %s\n", path)
	return nil
}

// defaultConfig returns the full default configuration with the given device name.
func defaultConfig(deviceName string) map[string]interface{} {
	return map[string]interface{}{
		"theme": "catppuccin-mocha",
		"client_id_command": map[string]interface{}{
			"command": "op",
			"args":    []string{"item", "get", "spotify-client", "--fields", "label=password", "--reveal"},
		},
		"client_port":                     8080,
		"login_redirect_uri":              "http://127.0.0.1:8989/login",
		"playback_format":                 "{status} {track} • {artists} {liked}\n{album} • {genres}\n{metadata}",
		"playback_metadata_fields":        []string{"repeat", "shuffle", "volume", "device"},
		"notify_timeout_in_secs":          0,
		"tracks_playback_limit":           50,
		"app_refresh_duration_in_ms":      32,
		"playback_refresh_duration_in_ms": 0,
		"page_size_in_rows":               20,
		"play_icon":                       "▶︎",
		"pause_icon":                      "⏸︎",
		"liked_icon":                      "👍🏼",
		"explicit_icon":                   "🍑",
		"border_type":                     "Rounded",
		"progress_bar_type":               "Rectangle",
		"progress_bar_position":           "Bottom",
		"genre_num":                       2,
		"cover_img_length":                9,
		"cover_img_width":                 5,
		"cover_img_scale":                 1.0,
		"cover_img_pixels":                64,
		"enable_media_control":            true,
		"enable_streaming":                "Always",
		"enable_audio_visualization":      false,
		"enable_notify":                   false,
		"enable_cover_image_cache":        true,
		"default_device":                  deviceName,
		"notify_streaming_only":           false,
		"seek_duration_secs":              5,
		"sort_artist_albums_by_type":      false,
		"volume_scroll_step":              5,
		"enable_mouse_scroll_volume":      true,
		"notify_format": map[string]interface{}{
			"summary": "{track} • {artists}",
			"body":    "{album}",
		},
		"layout": map[string]interface{}{
			"playback_window_position": "Top",
			"playback_window_height":   8,
			"library": map[string]interface{}{
				"playlist_percent": 40,
				"album_percent":    40,
			},
		},
		"device": map[string]interface{}{
			"name":          deviceName,
			"device_type":   "computer",
			"volume":        100,
			"bitrate":       320,
			"audio_cache":   true,
			"normalization": false,
			"autoplay":      false,
		},
	}
}
