package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/deusxyz/doomcode/editor/theme"
)

// The config file holds the general settings: fonts, palette, window
// size and the like. It lives next to keys, theme and fmt in the
// configuration directory (configFilePath). Each line is "name value";
// blank lines and # comments are ignored.
//
// A value is taken from, in order: a command-line flag that was given,
// the config file, then the code's default. Fonts have one more step
// between the last two: the fonts the palette is designed with.
//
// See docs/09-config-spec.md in the doomcode repository.

type settingDef struct {
	name string
	def  string // default, as written in the file
	flag string // command-line flag that overrides it, "" if none
	env  string // environment variable that overrides it, "" if none
	doc  string
}

var settingDefs = []settingDef{
	{"palette", theme.DefaultPaletteName, "palette", "", "colour palette: doom-light, doom-dark, acme, vampira, solarizedlight, solarizeddark"},
	{"font", "", "f", "", "proportional font; empty: the palette's font on macOS, else Lucida from plan9port"},
	{"font.fixed", "", "F", "", "fixed-width font; empty: as for font"},
	{"window", "1024x768", "W", "", "window size and position: WidthxHeight[@X,Y]"},
	{"columns", "2", "c", "", "number of columns at startup"},
	{"autoindent", "off", "a", "", "autoindent in every window: on or off"},
	{"focus", "click", "b", "", "click: typing goes to the last clicked window; mouse: to the window under the mouse"},
	{"tabstop", "4", "", "tabstop", "tab width in characters"},
	{"tabexpand", "off", "", "tabexpand", "type spaces for Tab: on or off"},
}

func settingDefFor(name string) *settingDef {
	for i := range settingDefs {
		if settingDefs[i].name == name {
			return &settingDefs[i]
		}
	}
	return nil
}

func configSettingsPath() string { return configFilePath("config") }

// parseSettings reads "name value" lines. Unknown names and bad values
// are reported and skipped.
func parseSettings(r io.Reader) (map[string]string, []error) {
	vals := map[string]string{}
	var errs []error
	sc := bufio.NewScanner(r)
	for lineno := 1; sc.Scan(); lineno++ {
		line := sc.Text()
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, value, _ := strings.Cut(line, " ")
		value = strings.TrimSpace(value)
		if value == "" {
			errs = append(errs, fmt.Errorf("line %d: %s: missing value", lineno, name))
			continue
		}
		def := settingDefFor(name)
		if def == nil {
			errs = append(errs, fmt.Errorf("line %d: unknown setting %q", lineno, name))
			continue
		}
		if err := checkSetting(name, value); err != nil {
			errs = append(errs, fmt.Errorf("line %d: %s: %v", lineno, name, err))
			continue
		}
		vals[name] = value
	}
	if err := sc.Err(); err != nil {
		errs = append(errs, err)
	}
	return vals, errs
}

func checkSetting(name, value string) error {
	switch name {
	case "palette":
		if _, ok := theme.PaletteByName(value); !ok {
			return fmt.Errorf("unknown palette %q", value)
		}
	case "columns", "tabstop":
		if n, err := strconv.Atoi(value); err != nil || n < 1 {
			return fmt.Errorf("want a positive number, got %q", value)
		}
	case "autoindent", "tabexpand":
		if _, err := parseOnOff(value); err != nil {
			return err
		}
	case "focus":
		if value != "click" && value != "mouse" {
			return fmt.Errorf("want click or mouse, got %q", value)
		}
	case "window":
		if value == "" {
			return fmt.Errorf("want WidthxHeight")
		}
	}
	return nil
}

func parseOnOff(v string) (bool, error) {
	switch v {
	case "on", "true", "yes":
		return true, nil
	case "off", "false", "no":
		return false, nil
	}
	return false, fmt.Errorf("want on or off, got %q", v)
}

// loadSettings reads the config file at path; a missing file is empty.
func loadSettings(path string) (map[string]string, []error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return map[string]string{}, []error{err}
	}
	defer f.Close()
	return parseSettings(f)
}

// configPaletteSet records that the config file chose the palette, so
// that it wins over the theme file's "palette" line as -palette does.
var configPaletteSet bool

// applySettings sets the flag variables and globals from the config
// file values, except where a flag (or for tabs, an environment
// variable) was given. flagSet reports whether a flag was given.
func applySettings(vals map[string]string, flagSet func(string) bool) {
	given := func(name string) (string, bool) {
		d := settingDefFor(name)
		v, ok := vals[name]
		if !ok || (d.flag != "" && flagSet(d.flag)) || (d.env != "" && os.Getenv(d.env) != "") {
			return "", false
		}
		return v, true
	}
	if v, ok := given("palette"); ok {
		*paletteName = v
		configPaletteSet = true
	}
	if v, ok := given("font"); ok {
		*varfontflag = v
	}
	if v, ok := given("font.fixed"); ok {
		*fixedfontflag = v
	}
	if v, ok := given("window"); ok {
		*winsize = v
	}
	if v, ok := given("columns"); ok {
		*ncol, _ = strconv.Atoi(v)
	}
	if v, ok := given("autoindent"); ok {
		*globalAutoIndent, _ = parseOnOff(v)
	}
	if v, ok := given("focus"); ok {
		*barflag = v == "click"
	}
	if v, ok := given("tabstop"); ok {
		n, _ := strconv.Atoi(v)
		global.maxtab = uint(n)
	}
	if v, ok := given("tabexpand"); ok {
		global.tabexpand, _ = parseOnOff(v)
	}
}

// startupPaletteName is the palette this run uses: -palette, else the
// config file, else the theme file's palette line, else the default.
func startupPaletteName(flagSet func(string) bool) string {
	if flagSet("palette") || configPaletteSet {
		return *paletteName
	}
	if tf, _ := readThemeFile(themeFilePath()); tf.Palette != "" {
		if _, ok := theme.PaletteByName(tf.Palette); ok {
			return tf.Palette
		}
	}
	return *paletteName
}

// applyPaletteFonts uses the palette's fonts where neither a flag nor
// the config file chose one. They are macOS system fonts, so other
// systems keep the built-in default.
func applyPaletteFonts(name string, vals map[string]string, flagSet func(string) bool, goos string) {
	_, varInConfig := vals["font"]
	_, fixedInConfig := vals["font.fixed"]
	paletteFonts.varFont = goos == "darwin" && !flagSet("f") && !varInConfig
	paletteFonts.fixedFont = goos == "darwin" && !flagSet("F") && !fixedInConfig
	p, ok := theme.PaletteByName(name)
	if !ok || goos != "darwin" {
		return
	}
	if _, inConfig := vals["font"]; p.VarFont != "" && !flagSet("f") && !inConfig {
		*varfontflag = p.VarFont
	}
	if _, inConfig := vals["font.fixed"]; p.FixedFont != "" && !flagSet("F") && !inConfig {
		*fixedfontflag = p.FixedFont
	}
}

// startupSettings loads the config file and applies it with the
// palette's fonts. Called after flag.Parse and before the display opens.
func startupSettings() {
	path := configSettingsPath()
	vals, errs := loadSettings(path)
	for _, err := range errs {
		log.Printf("config file %s: %v", path, err)
	}
	flagSet := func(name string) bool {
		set := false
		visitFlags(func(n string) {
			if n == name {
				set = true
			}
		})
		return set
	}
	applySettings(vals, flagSet)
	applyPaletteFonts(startupPaletteName(flagSet), vals, flagSet, runtime.GOOS)
}

// settingsDoc lists every setting with its default, in the file syntax.
func settingsDoc() string {
	var sb strings.Builder
	for _, d := range settingDefs {
		v := d.def
		if v == "" {
			v = "#"
		}
		fmt.Fprintf(&sb, "%-11s %-10s # %s\n", d.name, v, d.doc)
	}
	return sb.String()
}
