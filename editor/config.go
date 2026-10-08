package main

import (
	"os"
	"path/filepath"
	"strings"
)

// configName names doomcode's configuration: the directory that holds the
// keys, theme and fmt files and the prefix of the variables that point at
// them one by one.
const configName = "doomcode"

// configFilePath returns where the configuration file name lives:
// $DOOMCODE_<NAME> if set, otherwise $XDG_CONFIG_HOME/doomcode/<name>,
// otherwise $HOME/.config/doomcode/<name>.
func configFilePath(name string) string {
	if p := os.Getenv(strings.ToUpper(configName + "_" + name)); p != "" {
		return p
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, configName, name)
	}
	home := os.Getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, ".config", configName, name)
}
