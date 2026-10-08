// Package config holds doomcode's example configuration: one file per
// file of ~/.config/doomcode, each spelling out the built-in defaults.
// The editor embeds them so that "Config init" and "Config reset" work
// without the repository at hand. Tests in the editor check that the
// examples stay equal to the defaults in the code.
package config

import "embed"

// Files holds config, keys, theme and fmt.
//
//go:embed config keys theme fmt
var Files embed.FS

// Names lists the example files in the order they are shown.
var Names = []string{"config", "keys", "theme", "fmt"}
