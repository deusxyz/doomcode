package main

import (
	"log"
	"os"
	"strings"

	"github.com/deusxyz/doomcode/editor/theme"
)

// The theme file: $DOOMCODE_THEME, else $XDG_CONFIG_HOME/doomcode/theme, else
// $HOME/.config/doomcode/theme. Its format is described in theme/themefile.go.
//
// At startup the file's palette slots and styles are applied to the
// chosen palette before any window exists. "Theme reload" re-reads the
// file and re-applies the styles to the open windows; palette slot
// changes need a restart, because every frame copied its colours at
// creation.
//
// See docs/05-style-spec.md section 4 in the justcode project.

func themeFilePath() string { return configFilePath("theme") }

// readThemeFile parses the theme file at path. A missing file gives an
// empty theme and no error.
func readThemeFile(path string) (theme.ThemeFile, []error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return theme.ThemeFile{}, nil
		}
		return theme.ThemeFile{}, []error{err}
	}
	defer f.Close()
	return theme.ParseTheme(f)
}

// startupPalette picks the palette for this run: the dump file's if any,
// else the -palette flag if given explicitly, else the theme file's
// "palette" line, else the default; then applies the theme file to it.
// paletteName is the flag value and flagSet whether the user set it.
func startupPalette(base theme.Palette, baseIsDump bool, paletteName string, flagSet bool, path string) theme.Palette {
	tf, errs := readThemeFile(path)
	for _, err := range errs {
		log.Printf("theme file %s: %v", path, err)
	}
	p := base
	if !baseIsDump && !flagSet && tf.Palette != "" {
		if named, ok := theme.PaletteByName(tf.Palette); ok {
			p = named
		} else {
			log.Printf("theme file %s: unknown palette %q", path, tf.Palette)
		}
	}
	tf.Apply(&p)
	return p
}

// reloadThemeStyles re-reads the theme file and re-applies its styles on
// top of the built-in styles of the palette named base, then repaints
// every body. It returns the number of style lines applied and errors.
func reloadThemeStyles(path string) (int, []error) {
	tf, errs := readThemeFile(path)
	// Start again from the built-in styles of the palette in use, so
	// removing a line from the file takes effect too.
	name := *paletteName
	if tf.Palette != "" && !paletteFlagSet() {
		name = tf.Palette
	}
	builtin, ok := theme.PaletteByName(name)
	if !ok {
		builtin = theme.Light
	}
	global.palette.Styles = builtin.Styles
	only := theme.ThemeFile{Styles: tf.Styles}
	only.Apply(&global.palette)
	global.styles = nil // indices are re-allocated against the new styles
	global.row.AllWindows(func(w *Window) {
		t := &w.body
		if t.fr == nil {
			return
		}
		t.Restyle(t.org, t.org+t.fr.GetFrameFillStatus().Nchars)
	})
	return len(tf.Styles), errs
}

// paletteFlagSet reports whether -palette was given on the command line.
func paletteFlagSet() bool {
	set := false
	visitFlags(func(name string) {
		if name == "palette" {
			set = true
		}
	})
	return set
}

// themeCmd implements the Theme command:
//
//	Theme          list the styles in effect and the theme file path
//	Theme reload   re-read the theme file (styles only)
//	Theme file     print the theme file path
//	Theme slots    list the palette slots the file may set
func themeCmd(_ *Text, _ *Text, argt *Text, _, _ bool, arg string) {
	themeCommand(argt, arg)
}

// Reached through a variable set in init for the same initialization
// order reason as the Keys command (see keysfile.go).
var themeCommand func(argt *Text, arg string)

func init() {
	themeCommand = themeCommandImpl
}

func themeCommandImpl(argt *Text, arg string) {
	if r, _ := getarg(argt, false, true); r != "" {
		arg = r
	}
	arg = strings.TrimSpace(arg)
	path := themeFilePath()
	switch arg {
	case "":
		warning(nil, "Theme: styles (file %s):\n%s", path, global.palette.StylesDoc())
	case "reload":
		n, errs := reloadThemeStyles(path)
		for _, err := range errs {
			warning(nil, "Theme: %s: %v\n", path, err)
		}
		warning(nil, "Theme: %s: %d style line(s) applied; palette slots need a restart\n", path, n)
	case "file":
		warning(nil, "%s\n", path)
	case "slots":
		warning(nil, "Theme: palette slots: %s\n", strings.Join(theme.SlotNames(), " "))
	default:
		warning(nil, "Theme: unknown argument %q; use reload, file or slots\n", arg)
	}
}
