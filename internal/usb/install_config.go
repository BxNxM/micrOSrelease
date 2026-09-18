package usb

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strconv"
	"strings"
)

const installConfigVersion = 1

// InstallConfig describes how a firmware image is installed. Keeping this file
// beside each board's firmware makes adding a board a data-only change.
type InstallConfig struct {
	Version         int        `json:"version"`
	Protocol        string     `json:"protocol"`
	Chip            string     `json:"chip"`
	Platform        string     `json:"platform,omitempty"`
	InitialBaud     int        `json:"initial_baud"`
	FlashBaud       int        `json:"flash_baud"`
	FlashOffset     string     `json:"flash_offset"`
	EraseFlash      bool       `json:"erase_flash"`
	Compress        bool       `json:"compress"`
	ResetMode       string     `json:"reset_mode"`
	ResetAfterFlash bool       `json:"reset_after_flash"`
	FlashMode       string     `json:"flash_mode,omitempty"`
	FlashFrequency  string     `json:"flash_frequency,omitempty"`
	FlashSize       string     `json:"flash_size,omitempty"`
	Hint            string     `json:"hint,omitempty"`
	REPL            REPLConfig `json:"repl,omitempty"`
}

// REPLConfig controls MicroPython filesystem access after the firmware boots.
type REPLConfig struct {
	Baud                    int              `json:"baud,omitempty"`
	ConnectTimeoutSeconds   int              `json:"connect_timeout_seconds,omitempty"`
	ReconnectTimeoutSeconds int              `json:"reconnect_timeout_seconds,omitempty"`
	ConfigPaths             []string         `json:"config_paths,omitempty"`
	RestorePath             string           `json:"restore_path,omitempty"`
	Resources               []UpdateResource `json:"resources,omitempty"`
}

// UpdateResource maps an embedded file to a path on the MicroPython filesystem.
type UpdateResource struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

func loadInstallConfig(files fs.FS, image Image) (InstallConfig, uint32, error) {
	if files == nil {
		return InstallConfig{}, 0, fmt.Errorf("firmware filesystem is unavailable")
	}
	configPath := path.Join(path.Dir(image.Path), "install.json")
	file, err := files.Open(configPath)
	if err != nil {
		return InstallConfig{}, 0, fmt.Errorf("open install config %s: %w", configPath, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var config InstallConfig
	if err := decoder.Decode(&config); err != nil {
		return InstallConfig{}, 0, fmt.Errorf("decode install config %s: %w", configPath, err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return InstallConfig{}, 0, fmt.Errorf("decode install config %s: %w", configPath, err)
	}
	if config.Platform == "" {
		config.Platform = normalizeChip(config.Chip)
	}
	offset, err := validateInstallConfig(config)
	if err != nil {
		return InstallConfig{}, 0, fmt.Errorf("invalid install config %s: %w", configPath, err)
	}
	config.REPL = config.REPL.withDefaults()
	return config, offset, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func validateInstallConfig(config InstallConfig) (uint32, error) {
	if config.Version != installConfigVersion {
		return 0, fmt.Errorf("unsupported version %d", config.Version)
	}
	if config.Protocol != "esp-rom" {
		return 0, fmt.Errorf("unsupported protocol %q", config.Protocol)
	}
	if !supportedESPChip(config.Chip) {
		return 0, fmt.Errorf("unsupported chip %q", config.Chip)
	}
	if strings.ContainsAny(config.Platform, `/\\`) || config.Platform == "." || config.Platform == ".." {
		return 0, fmt.Errorf("invalid platform %q", config.Platform)
	}
	if config.InitialBaud <= 0 || config.FlashBaud <= 0 {
		return 0, fmt.Errorf("initial_baud and flash_baud must be positive")
	}
	if !supportedResetMode(config.ResetMode) {
		return 0, fmt.Errorf("unsupported reset_mode %q", config.ResetMode)
	}
	if err := validateREPLConfig(config.REPL.withDefaults()); err != nil {
		return 0, err
	}
	offset, err := strconv.ParseUint(config.FlashOffset, 0, 32)
	if err != nil {
		return 0, fmt.Errorf("flash_offset %q: %w", config.FlashOffset, err)
	}
	return uint32(offset), nil
}

func (config REPLConfig) withDefaults() REPLConfig {
	if config.Baud <= 0 {
		config.Baud = 115200
	}
	if config.ConnectTimeoutSeconds <= 0 {
		config.ConnectTimeoutSeconds = 10
	}
	if len(config.ConfigPaths) == 0 {
		config.ConfigPaths = []string{"/config/node_config.json", "/node_config.json"}
	}
	if config.RestorePath == "" {
		config.RestorePath = "/config/node_config.json"
	}
	return config
}

func validateREPLConfig(config REPLConfig) error {
	if config.ReconnectTimeoutSeconds < 0 {
		return fmt.Errorf("reconnect_timeout_seconds must be zero (wait until cancelled) or positive")
	}
	for _, name := range append(append([]string(nil), config.ConfigPaths...), config.RestorePath) {
		if !validDevicePath(name) {
			return fmt.Errorf("invalid MicroPython path %q", name)
		}
	}
	for _, resource := range config.Resources {
		if !fs.ValidPath(resource.Source) || !strings.HasPrefix(resource.Source, "modules/") {
			return fmt.Errorf("invalid update resource source %q", resource.Source)
		}
		if !validDevicePath(resource.Target) {
			return fmt.Errorf("invalid update resource target %q", resource.Target)
		}
	}
	return nil
}

func validDevicePath(name string) bool {
	return strings.HasPrefix(name, "/") && path.Clean(name) == name && name != "/"
}

func supportedResetMode(mode string) bool {
	switch mode {
	case "default", "usb-jtag", "no-reset", "auto":
		return true
	default:
		return false
	}
}
