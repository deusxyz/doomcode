package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deusxyz/doomcode/editor/theme"
)

func TestUserThemes(t *testing.T) {
	dir := t.TempDir()
	write := func(name, s string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("mint", "palette doom-dark\nstyle keyword fg=#00ff00\ntext.back #102010\nfont /mnt/font/X/12a/font\n")
	write("plain", "style comment -\n")         // base defaults to doom-light
	write("acme", "style keyword fg=#000000\n") // shadows a built-in: refused
	write("odd", "palette mint\n")              // base must be built in
	write(".hidden", "palette doom-dark\n")     // skipped
	write("mint.bak", "palette doom-dark\n")    // skipped

	ps, errs := loadUserThemes(dir)
	if len(errs) != 2 {
		t.Errorf("errors %v; want 2 (built-in name, base not built in)", errs)
	}
	if len(ps) != 2 {
		t.Fatalf("palettes %v; want mint and plain", ps)
	}
	mint := ps["mint"]
	if s, _ := mint.Styles.Resolve("keyword"); s.Fg.Color != 0x00FF00FF {
		t.Errorf("mint keyword %+v", s)
	}
	if s, _ := mint.Styles.Resolve("string"); s != theme.DoomDark.Styles["string"] {
		t.Errorf("mint does not inherit doom-dark's string style: %+v", s)
	}
	if mint.Text.Back.Color != 0x102010FF || mint.VarFont != "/mnt/font/X/12a/font" || mint.FixedFont != theme.DoomDark.FixedFont {
		t.Errorf("mint back %#x fonts %q %q", mint.Text.Back.Color, mint.VarFont, mint.FixedFont)
	}
	if _, ok := ps["plain"].Styles.Resolve("comment"); ok {
		t.Error("plain still has a comment style")
	}

	defer theme.RegisterPalettes(nil)
	theme.RegisterPalettes(ps)
	if _, ok := theme.PaletteByName("mint"); !ok {
		t.Error("mint not found after registering")
	}
	if names := strings.Join(theme.PaletteNames(), " "); !strings.HasSuffix(names, "mint plain") {
		t.Errorf("palette names %q", names)
	}
	if err := checkSetting("palette", "mint"); err != nil {
		t.Errorf("config rejects a registered palette: %v", err)
	}
}

func TestThemeFileFonts(t *testing.T) {
	tf, errs := theme.ParseTheme(strings.NewReader("font /mnt/font/A/14a/font\nfont.fixed /mnt/font/B/13a/font\nfont\n"))
	if len(errs) != 1 || tf.VarFont != "/mnt/font/A/14a/font" || tf.FixedFont != "/mnt/font/B/13a/font" {
		t.Errorf("fonts %q %q errs %v", tf.VarFont, tf.FixedFont, errs)
	}
}
