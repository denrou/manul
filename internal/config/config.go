// Package config loads manul's user configuration from
// ${XDG_CONFIG_HOME:-~/.config}/manul/config.toml.
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds the user-tunable settings. Zero values are replaced by
// Defaults in Load; a partial config file overrides only the keys it sets.
type Config struct {
	Theme     string `toml:"theme"`
	MaxWidth  int    `toml:"max_width"`
	StylePath string `toml:"style_path"`
	Mouse     bool   `toml:"mouse"`
}

// Defaults returns the configuration used when no config file exists.
// Mouse defaults to false because cell-motion capture breaks
// terminal-native text selection in a reader app.
func Defaults() Config {
	return Config{
		Theme:    "auto",
		MaxWidth: 100,
		Mouse:    false,
	}
}

// Dir returns manul's configuration directory,
// ${XDG_CONFIG_HOME:-~/.config}/manul. The XDG_CONFIG_HOME override is
// honored on every platform so tests can point it at a temp directory.
func Dir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "manul"), nil
}

// Load reads <Dir()>/config.toml. An absent file yields Defaults; a
// partial file merges over Defaults; invalid TOML is an error.
func Load() (Config, error) {
	cfg := Defaults()
	dir, err := Dir()
	if err != nil {
		return cfg, err
	}
	path := filepath.Join(dir, "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Defaults(), err
	}
	return cfg, nil
}
