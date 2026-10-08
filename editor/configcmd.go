package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/deusxyz/doomcode/config"
)

// The Config command and the config action (Cmd-, and Ctrl-B ,):
//
//	Config           open the configuration directory, creating it
//	Config init      copy the example files where none exist yet
//	Config reset F   replace file F with its example, keeping F.bak
//	Config reload    re-read config, keys, theme and fmt
//	Config file      print the directory
//	Config defaults  print every setting with its default
//
// See docs/09-config-spec.md section 4 in the doomcode repository.

// configDir is the directory that holds config, keys, theme and fmt.
func configDir() string { return filepath.Dir(configFilePath("config")) }

func configCmd(et *Text, _ *Text, argt *Text, _, _ bool, arg string) {
	configCommand(et, argt, arg)
}

// Reached through a variable set in init for the same initialization
// order reason as the Keys command (see keysfile.go).
var configCommand func(et *Text, argt *Text, arg string)

func init() {
	configCommand = configCommandImpl
}

func configCommandImpl(et *Text, argt *Text, arg string) {
	if r, _ := getarg(argt, false, true); r != "" {
		arg = r
	}
	words := strings.Fields(arg)
	dir := configDir()
	switch {
	case len(words) == 0:
		configOpen(et)
	case words[0] == "init" && len(words) == 1:
		made, err := configInit(dir)
		if err != nil {
			warning(nil, "Config: %v\n", err)
		}
		if len(made) == 0 {
			warning(nil, "Config: %s: every file already exists; nothing copied\n", dir)
		} else {
			warning(nil, "Config: %s: copied the examples %s\n", dir, strings.Join(made, ", "))
		}
	case words[0] == "reset" && len(words) == 2:
		if err := configReset(dir, words[1]); err != nil {
			warning(nil, "Config: %v\n", err)
			return
		}
		warning(nil, "Config: %s reset to the example; the old one is %s.bak\n", filepath.Join(dir, words[1]), words[1])
	case words[0] == "reload" && len(words) == 1:
		configReload()
	case words[0] == "file" && len(words) == 1:
		warning(nil, "%s\n", dir)
	case words[0] == "defaults" && len(words) == 1:
		warning(nil, "Config: settings and their defaults (file %s):\n%s", configSettingsPath(), settingsDoc())
	default:
		warning(nil, "Config: unknown arguments %q; use init, reset FILE, reload, file or defaults\n", arg)
	}
}

// configOpen opens the configuration directory in a window, creating it
// first; Ctrl-O on a file name in it opens the file.
func configOpen(t *Text) {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		warning(nil, "Config: %v\n", err)
		return
	}
	if t == nil {
		t = global.focusText()
	}
	openfile(t, &Expand{name: dir + string(filepath.Separator), jump: true})
	if entries, err := os.ReadDir(dir); err == nil && len(entries) == 0 {
		warning(nil, "Config: %s is empty, so the built-in defaults apply. Config init copies the example files there to edit.\n", dir)
	}
}

// configInit copies each example into dir unless the file exists, and
// returns the names copied.
func configInit(dir string) ([]string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var made []string
	for _, name := range config.Names {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		b, err := config.Files.ReadFile(name)
		if err != nil {
			return made, err
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			return made, err
		}
		made = append(made, name)
	}
	return made, nil
}

// configReset replaces dir/name with its example, saving the old file
// as name.bak.
func configReset(dir, name string) error {
	b, err := config.Files.ReadFile(name)
	if err != nil {
		return fmt.Errorf("no example named %q; examples: %s", name, strings.Join(config.Names, " "))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, name)
	if old, err := os.ReadFile(path); err == nil {
		if err := os.WriteFile(path+".bak", old, 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(path, b, 0o644)
}

// configReload re-reads every file. Settings that need a restart (fonts,
// palette, window) say so.
func configReload() {
	keysCommand(nil, nil, nil, false, false, "reload")
	themeCommand(nil, "reload")
	fmtCmd(nil, nil, nil, false, false, "reload")
	path := configSettingsPath()
	vals, errs := loadSettings(path)
	for _, err := range errs {
		warning(nil, "Config: %s: %v\n", path, err)
	}
	if v, ok := vals["tabstop"]; ok && os.Getenv("tabstop") == "" {
		fmt.Sscan(v, &global.maxtab)
	}
	if v, ok := vals["tabexpand"]; ok && os.Getenv("tabexpand") == "" {
		global.tabexpand, _ = parseOnOff(v)
	}
	warning(nil, "Config: %s: %d setting(s) read; palette, fonts, window and columns take effect after a restart\n", path, len(vals))
}
