package main

import (
	"testing"

	"github.com/deusxyz/doomcode/editor/dumpfile"
	"github.com/deusxyz/doomcode/editor/theme"
)

func TestSwitchPalette(t *testing.T) {
	MakeWindowScaffold(&dumpfile.Content{
		Columns: []dumpfile.Column{{}},
		Windows: []*dumpfile.Window{{Column: 0, Tag: dumpfile.Text{Buffer: "/a/b Del"}, Body: dumpfile.Text{Buffer: "x"}}},
	})
	if global.row.display == nil {
		t.Skip("scaffold has no display")
	}
	savedName, savedPalette := *paletteName, global.palette
	savedVar, savedFixed, savedTag, savedFollow := *varfontflag, *fixedfontflag, global.tagfont, paletteFonts
	defer func() {
		*paletteName, global.palette, global.styles = savedName, savedPalette, nil
		*varfontflag, *fixedfontflag, global.tagfont, paletteFonts = savedVar, savedFixed, savedTag, savedFollow
	}()
	paletteFonts.varFont, paletteFonts.fixedFont = false, false // fonts stay; colours are the subject
	t.Setenv("DOOMCODE_THEME", t.TempDir()+"/none")             // no theme file on top

	if global.switchPalette("no-such-palette") {
		t.Fatal("unknown palette switched")
	}
	if !global.switchPalette("doom-dark") {
		t.Fatal("doom-dark did not switch")
	}
	if *paletteName != "doom-dark" {
		t.Errorf("palette name %q", *paletteName)
	}
	if got, want := global.palette.Text.Back.Color, theme.DoomDark.Text.Back.Color; got != want {
		t.Errorf("body background %#x; want doom-dark's %#x", got, want)
	}
	if global.palette.StylesDoc() != theme.DoomDark.StylesDoc() {
		t.Error("styles are not doom-dark's")
	}
}

func TestPaletteNames(t *testing.T) {
	names := theme.PaletteNames()
	if len(names) != 6 || names[0] != "doom-light" || names[1] != "doom-dark" {
		t.Errorf("palette names %v", names)
	}
}
