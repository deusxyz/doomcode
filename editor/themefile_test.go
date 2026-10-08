package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/deusxyz/doomcode/editor/theme"
)

func writeTheme(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "theme")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestStartupPalette(t *testing.T) {
	path := writeTheme(t, "palette vampira\nstyle keyword fg=#123456\ntext.back #101010\n")

	// No flag: the file picks the palette, then its overrides apply.
	p := startupPalette(theme.Light, false, "acme", false, path)
	if p.Tag.Back != theme.Dark.Tag.Back {
		t.Errorf("file's palette line ignored: tag.back %+v", p.Tag.Back)
	}
	if p.Text.Back.Color != 0x101010FF {
		t.Errorf("text.back override lost: %+v", p.Text.Back)
	}
	if s, _ := p.Styles.Resolve("keyword"); s.Fg.Color != 0x123456FF {
		t.Errorf("keyword override lost: %+v", s)
	}
	if s, _ := p.Styles.Resolve("comment"); s != theme.Dark.Styles["comment"] {
		t.Errorf("other styles should come from vampira: %+v", s)
	}

	// -palette given: the flag wins over the file's palette line.
	p = startupPalette(theme.Light, false, "acme", true, path)
	if p.Tag.Back != theme.Light.Tag.Back {
		t.Errorf("flag should win over the file: tag.back %+v", p.Tag.Back)
	}
	if p.Text.Back.Color != 0x101010FF {
		t.Errorf("overrides still apply with the flag: %+v", p.Text.Back)
	}

	// A dump file's palette is kept as is, with overrides applied.
	p = startupPalette(theme.SolarizedDark, true, "acme", false, path)
	if p.Tag.Back != theme.SolarizedDark.Tag.Back {
		t.Errorf("dump palette replaced: %+v", p.Tag.Back)
	}

	// Missing file: nothing changes.
	p = startupPalette(theme.Light, false, "acme", false, filepath.Join(t.TempDir(), "none"))
	if p.Text.Back != theme.Light.Text.Back || len(p.Styles) != len(theme.Light.Styles) {
		t.Errorf("missing theme file changed the palette")
	}
}

func TestReloadThemeStyles(t *testing.T) {
	_, w := makeFocusScaffold()
	body := &w[0].body
	fr := &recordingFrame{nchars: 2}
	body.fr = fr
	body.file.Styles().Set(0, 2, "keyword")
	global.palette = theme.Light
	global.styles = nil
	defer func() {
		global.palette = theme.Light
		global.styles = nil
	}()

	path := writeTheme(t, "style keyword fg=#123456 bg=#654321\nstyle comment -\n")
	n, errs := reloadThemeStyles(path)
	if n != 2 || len(errs) != 0 {
		t.Fatalf("n=%d errs=%v", n, errs)
	}
	if s, _ := global.palette.Styles.Resolve("keyword"); s.Fg.Color != 0x123456FF || s.Bg.Color != 0x654321FF {
		t.Errorf("keyword not applied: %+v", s)
	}
	if _, ok := global.palette.Styles.Resolve("comment"); ok {
		t.Error("comment not removed")
	}
	if len(fr.restyles) != 1 || fr.restyles[0].p0 != 0 || fr.restyles[0].p1 != 2 {
		t.Errorf("visible body not restyled: %+v", fr.restyles)
	}
	// A new style set was built against the new styles.
	if global.styles == nil || fr.table == nil {
		t.Errorf("style set or frame table missing after reload")
	}

	// Reloading an emptier file restores the built-in styles.
	path = writeTheme(t, "")
	reloadThemeStyles(path)
	if s, _ := global.palette.Styles.Resolve("keyword"); s != theme.Light.Styles["keyword"] {
		t.Errorf("built-in keyword not restored: %+v", s)
	}
	if _, ok := global.palette.Styles.Resolve("comment"); !ok {
		t.Error("built-in comment not restored")
	}
}

func TestThemeFilePath(t *testing.T) {
	t.Setenv("DOOMCODE_THEME", "/x/theme")
	t.Setenv("XDG_CONFIG_HOME", "/xdg")
	t.Setenv("HOME", "/home")
	if got := themeFilePath(); got != "/x/theme" {
		t.Errorf("DOOMCODE_THEME: %q", got)
	}
	t.Setenv("DOOMCODE_THEME", "")
	if got := themeFilePath(); got != filepath.Join("/xdg", "doomcode", "theme") {
		t.Errorf("XDG: %q", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	if got := themeFilePath(); got != filepath.Join("/home", ".config", "doomcode", "theme") {
		t.Errorf("HOME: %q", got)
	}
}
