package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deusxyz/doomcode/config"
	"github.com/deusxyz/doomcode/editor/theme"
)

func example(t *testing.T, name string) string {
	t.Helper()
	b, err := config.Files.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The examples spell out the defaults: applying one changes nothing.

func TestExampleKeysEqualDefaults(t *testing.T) {
	km, pkm := Keymap{}, Keymap{}
	if _, errs := loadKeysText(km, pkm, strings.NewReader(example(t, "keys"))); len(errs) > 0 {
		t.Fatalf("example keys: %v", errs)
	}
	if got, want := km.String(), DefaultKeymap().String(); got != want {
		t.Errorf("config/keys differs from the default keymap; regenerate it.\ngot:\n%s\nwant:\n%s", got, want)
	}
	if got, want := pkm.String(), DefaultPrefixKeymap().String(); got != want {
		t.Errorf("config/keys prefix lines differ from the defaults.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestExampleThemeChangesNothing(t *testing.T) {
	tf, errs := theme.ParseTheme(strings.NewReader(example(t, "theme")))
	if len(errs) > 0 || tf.Palette != "" || len(tf.Styles) != 0 || len(tf.Slots) != 0 {
		t.Errorf("config/theme must be all comments: %+v %v", tf, errs)
	}
	// Its commented style lines are doom-light's styles.
	var lines []string
	for _, l := range strings.Split(example(t, "theme"), "\n") {
		if strings.HasPrefix(l, "# style ") {
			lines = append(lines, strings.TrimPrefix(l, "# "))
		}
	}
	tf, errs = theme.ParseTheme(strings.NewReader(strings.Join(lines, "\n")))
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	p := theme.DoomLight
	p.Styles = theme.Styles{}
	tf.Apply(&p)
	if got, want := p.StylesDoc(), theme.DoomLight.StylesDoc(); got != want {
		t.Errorf("config/theme styles differ from doom-light.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestExampleFmtEqualDefaults(t *testing.T) {
	tab := &fmtTable{}
	if _, errs := loadFmtText(tab, strings.NewReader(example(t, "fmt"))); len(errs) > 0 {
		t.Fatal(errs)
	}
	if got, want := tab.doc(), defaultFmtTable().doc(); got != want || !tab.onPut {
		t.Errorf("config/fmt: %q onPut=%v; want %q", got, tab.onPut, want)
	}
}

func TestExampleConfigEqualDefaults(t *testing.T) {
	src := example(t, "config")
	vals, errs := parseSettings(strings.NewReader(src))
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	for _, d := range settingDefs {
		v, ok := vals[d.name]
		switch {
		case ok && v != d.def:
			t.Errorf("config/config: %s = %q; default is %q", d.name, v, d.def)
		case !ok && d.def != "":
			t.Errorf("config/config lacks %s (default %q)", d.name, d.def)
		case !ok && !strings.Contains(src, "# "+d.name+" "):
			t.Errorf("config/config does not mention %s", d.name)
		}
	}
}

func TestParseSettings(t *testing.T) {
	vals, errs := parseSettings(strings.NewReader("palette doom-dark\nfont /mnt/font/X/12a/font # c\ncolumns 3\nnope 1\ncolumns x\nfocus mouse\nautoindent maybe\ntabstop\n"))
	if len(errs) != 4 {
		t.Errorf("errors %v; want 4 (unknown, bad columns, bad autoindent, missing value)", errs)
	}
	for k, v := range map[string]string{"palette": "doom-dark", "font": "/mnt/font/X/12a/font", "columns": "3", "focus": "mouse"} {
		if vals[k] != v {
			t.Errorf("%s = %q; want %q", k, vals[k], v)
		}
	}
}

func TestApplySettingsPrecedence(t *testing.T) {
	saved := []string{*paletteName, *varfontflag, *fixedfontflag, *winsize}
	savedCol, savedBar, savedAI, savedSet := *ncol, *barflag, *globalAutoIndent, configPaletteSet
	savedFollow := paletteFonts
	defer func() { paletteFonts = savedFollow }()
	defer func() {
		*paletteName, *varfontflag, *fixedfontflag, *winsize = saved[0], saved[1], saved[2], saved[3]
		*ncol, *barflag, *globalAutoIndent, configPaletteSet = savedCol, savedBar, savedAI, savedSet
	}()
	*varfontflag, *fixedfontflag = "flagfont", "default-fixed"
	vals := map[string]string{"palette": "doom-dark", "font": "configfont", "window": "800x600", "columns": "3", "focus": "mouse", "autoindent": "on"}
	given := map[string]bool{"f": true} // -f was on the command line
	applySettings(vals, func(n string) bool { return given[n] })
	if *varfontflag != "flagfont" {
		t.Errorf("a flag lost to the config: font %q", *varfontflag)
	}
	if *paletteName != "doom-dark" || !configPaletteSet || *winsize != "800x600" || *ncol != 3 || *barflag || !*globalAutoIndent {
		t.Errorf("config not applied: palette %q window %q columns %d focus-click %v autoindent %v", *paletteName, *winsize, *ncol, *barflag, *globalAutoIndent)
	}

	// The palette's fonts fill what neither a flag nor the config set,
	// on macOS only.
	*varfontflag, *fixedfontflag = "default-var", "default-fixed"
	applyPaletteFonts("doom-dark", map[string]string{}, func(string) bool { return false }, "linux")
	if *varfontflag != "default-var" {
		t.Errorf("palette font applied off macOS: %q", *varfontflag)
	}
	applyPaletteFonts("doom-dark", map[string]string{"font.fixed": "cf"}, func(string) bool { return false }, "darwin")
	if *varfontflag != theme.DoomDark.VarFont || *fixedfontflag != "default-fixed" {
		t.Errorf("palette fonts: %q %q; want the palette's var font and the fixed one untouched", *varfontflag, *fixedfontflag)
	}
}

func TestConfigInitAndReset(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "doomcode")
	made, err := configInit(dir)
	if err != nil || strings.Join(made, " ") != "config keys theme fmt" {
		t.Fatalf("init made %v, %v", made, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keys"), []byte("C-k -\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if made, _ := configInit(dir); len(made) != 0 {
		t.Errorf("second init overwrote %v", made)
	}
	if err := configReset(dir, "keys"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "keys.bak")); string(b) != "C-k -\n" {
		t.Errorf("backup %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "keys")); string(b) != example(t, "keys") {
		t.Error("reset did not restore the example")
	}
	if err := configReset(dir, "nope"); err == nil {
		t.Error("reset of an unknown file succeeded")
	}
}

func TestConfigKeys(t *testing.T) {
	for km, key := range map[*Keymap]string{ptrTo(DefaultKeymap()): "Cmd-,", ptrTo(DefaultPrefixKeymap()): ","} {
		r, err := ParseKey(key)
		if err != nil {
			t.Fatal(err)
		}
		if a := km.Lookup(r); a == nil || a.Name != "config" {
			t.Errorf("%s bound to %v; want config", key, a)
		}
	}
}

func ptrTo(km Keymap) *Keymap { return &km }
