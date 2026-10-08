package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/deusxyz/doomcode/editor/theme"
)

// Themes as files: every file in ~/.config/doomcode/themes is a palette
// named after the file. It uses the theme file format and starts from
// the built-in palette its "palette" line names (doom-light if none), so
// a theme lists only what it changes, or every slot and style for a
// palette of its own. Built-in names cannot be shadowed.
//
// See docs/09-config-spec.md section 6 in the doomcode repository.

func themesDir() string { return filepath.Join(configDir(), "themes") }

// loadUserThemes builds the palettes in dir. A missing dir is no error.
func loadUserThemes(dir string) (map[string]theme.Palette, []error) {
	ps := map[string]theme.Palette{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return ps, nil
		}
		return ps, []error{err}
	}
	var errs []error
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".bak") || strings.HasSuffix(name, "~") {
			continue
		}
		if theme.IsBuiltin(name) {
			errs = append(errs, fmt.Errorf("%s: a built-in palette has this name; rename the file", name))
			continue
		}
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		tf, perrs := theme.ParseTheme(f)
		f.Close()
		for _, pe := range perrs {
			errs = append(errs, fmt.Errorf("%s: %v", name, pe))
		}
		base := tf.Palette
		if base == "" {
			base = theme.DefaultPaletteName
		}
		if !theme.IsBuiltin(base) {
			errs = append(errs, fmt.Errorf("%s: base palette %q is not built in", name, base))
			continue
		}
		p, _ := theme.PaletteByName(base)
		tf.Apply(&p)
		ps[name] = p
	}
	return ps, errs
}

// registerUserThemes loads and registers the theme files, reporting
// errors through report.
func registerUserThemes(report func(error)) int {
	ps, errs := loadUserThemes(themesDir())
	for _, err := range errs {
		report(err)
	}
	theme.RegisterPalettes(ps)
	return len(ps)
}

func startupThemes() {
	registerUserThemes(func(err error) { log.Printf("themes %s: %v", themesDir(), err) })
}
