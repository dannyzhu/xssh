package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the full xssh configuration, loaded from ~/.xssh/config.yaml.
type Config struct {
	General GeneralConfig       `yaml:"general"`
	UI      UIConfig            `yaml:"ui"`
	Groups  map[string][]string `yaml:"groups"`
}

// GeneralConfig controls session behaviour.
type GeneralConfig struct {
	ScrollbackLines   int           `yaml:"scrollback_lines"`
	ReconnectAttempts int           `yaml:"reconnect_attempts"`
	ReconnectInterval time.Duration `yaml:"reconnect_interval"`
	SSHTimeout        time.Duration `yaml:"ssh_timeout"`
	BroadcastRealtime bool          `yaml:"broadcast_realtime"`
}

// UIConfig controls visual appearance.
type UIConfig struct {
	BorderStyle        string `yaml:"border_style"`
	FocusedColor       string `yaml:"focused_color"`
	InactiveColor      string `yaml:"inactive_color"`
	DisconnectedColor  string `yaml:"disconnected_color"`
	ReconnectingColor  string `yaml:"reconnecting_color"`
	BroadcastColor     string `yaml:"broadcast_color"`
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		General: GeneralConfig{
			ScrollbackLines:   5000,
			ReconnectAttempts: 3,
			ReconnectInterval: 5 * time.Second,
			SSHTimeout:        10 * time.Second,
			BroadcastRealtime: false,
		},
		UI: UIConfig{
			BorderStyle:       "rounded",
			FocusedColor:      "#00BFFF",
			InactiveColor:     "#555555",
			DisconnectedColor: "#FF4444",
			ReconnectingColor: "#FFB347",
			BroadcastColor:    "#FF6B6B",
		},
		Groups: make(map[string][]string),
	}
}

// configDir returns the ~/.xssh directory path.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".xssh"), nil
}

// configPath returns the full path to the config file.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads ~/.xssh/config.yaml and merges it with defaults.
// If the file does not exist, the default config is returned without error.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	path, err := configPath()
	if err != nil {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	// Unmarshal into a partial struct and merge (so missing keys keep defaults)
	var partial Config
	if err := yaml.Unmarshal(data, &partial); err != nil {
		return cfg, err
	}

	mergeConfig(cfg, &partial)
	return cfg, nil
}

// Save writes cfg to ~/.xssh/config.yaml, creating the directory if needed.
func Save(cfg *Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, "config.yaml")
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// SaveGroup adds or overwrites a named group in the config file.
func SaveGroup(name string, targets []string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	if cfg.Groups == nil {
		cfg.Groups = make(map[string][]string)
	}
	cfg.Groups[name] = targets
	return Save(cfg)
}

// LoadGroup returns the targets for a named group, or nil if not found.
func LoadGroup(name string) ([]string, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	targets, ok := cfg.Groups[name]
	if !ok {
		return nil, nil
	}
	return targets, nil
}

// ListGroups returns all group names from the config.
func ListGroups() (map[string][]string, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	return cfg.Groups, nil
}

// mergeConfig copies non-zero fields from partial into base.
func mergeConfig(base, partial *Config) {
	if partial.General.ScrollbackLines != 0 {
		base.General.ScrollbackLines = partial.General.ScrollbackLines
	}
	if partial.General.ReconnectAttempts != 0 {
		base.General.ReconnectAttempts = partial.General.ReconnectAttempts
	}
	if partial.General.ReconnectInterval != 0 {
		base.General.ReconnectInterval = partial.General.ReconnectInterval
	}
	if partial.General.SSHTimeout != 0 {
		base.General.SSHTimeout = partial.General.SSHTimeout
	}
	base.General.BroadcastRealtime = partial.General.BroadcastRealtime

	if partial.UI.BorderStyle != "" {
		base.UI.BorderStyle = partial.UI.BorderStyle
	}
	if partial.UI.FocusedColor != "" {
		base.UI.FocusedColor = partial.UI.FocusedColor
	}
	if partial.UI.InactiveColor != "" {
		base.UI.InactiveColor = partial.UI.InactiveColor
	}
	if partial.UI.DisconnectedColor != "" {
		base.UI.DisconnectedColor = partial.UI.DisconnectedColor
	}
	if partial.UI.ReconnectingColor != "" {
		base.UI.ReconnectingColor = partial.UI.ReconnectingColor
	}
	if partial.UI.BroadcastColor != "" {
		base.UI.BroadcastColor = partial.UI.BroadcastColor
	}

	if len(partial.Groups) > 0 {
		if base.Groups == nil {
			base.Groups = make(map[string][]string)
		}
		for k, v := range partial.Groups {
			base.Groups[k] = v
		}
	}
}
