package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deusxyz/doomcode/editor/dumpfile"
)

func TestResolveAutoPalette(t *testing.T) {
	savedName, savedSet, savedAuto := *paletteName, configPaletteSet, autoPalette
	defer func() { *paletteName, configPaletteSet, autoPalette = savedName, savedSet, savedAuto }()

	*paletteName = "doom-light"
	resolveAutoPalette(map[string]string{}, func() bool { return true })
	if autoPalette.on || *paletteName != "doom-light" {
		t.Errorf("a named palette turned auto on: %v %q", autoPalette.on, *paletteName)
	}

	*paletteName = "auto"
	resolveAutoPalette(map[string]string{"palette.dark": "solarizeddark"}, func() bool { return true })
	if !autoPalette.on || *paletteName != "solarizeddark" || autoPalette.light != "doom-light" {
		t.Errorf("auto, dark system: on %v palette %q light %q", autoPalette.on, *paletteName, autoPalette.light)
	}
	*paletteName = "auto"
	resolveAutoPalette(map[string]string{}, func() bool { return false })
	if *paletteName != "doom-light" {
		t.Errorf("auto, light system: palette %q", *paletteName)
	}
	if err := checkSetting("palette", "auto"); err != nil {
		t.Error(err)
	}
	if err := checkSetting("palette.dark", "auto"); err == nil {
		t.Error("palette.dark accepted auto")
	}
}

func TestAutoPaletteLoop(t *testing.T) {
	MakeWindowScaffold(&dumpfile.Content{
		Columns: []dumpfile.Column{{}},
		Windows: []*dumpfile.Window{{Column: 0, Tag: dumpfile.Text{Buffer: "/a/b Del"}, Body: dumpfile.Text{Buffer: "x"}}},
	})
	if global.row.display == nil {
		t.Skip("scaffold has no display")
	}
	savedName, savedPalette, savedAuto, savedPoll := *paletteName, global.palette, autoPalette, autoPalettePoll
	savedFollow := paletteFonts
	defer func() {
		*paletteName, global.palette, autoPalette, autoPalettePoll = savedName, savedPalette, savedAuto, savedPoll
		paletteFonts, global.styles = savedFollow, nil
	}()
	t.Setenv("DOOMCODE_THEME", t.TempDir()+"/none")
	paletteFonts.varFont, paletteFonts.fixedFont = false, false
	autoPalettePoll = 10 * time.Millisecond
	autoPalette.on, autoPalette.light, autoPalette.dark = true, "doom-light", "doom-dark"
	*paletteName = "doom-light"

	var dark atomic.Bool
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { autoPaletteLoop(ctx, global, dark.Load); close(done) }()
	defer func() { cancel(); <-done }()

	current := func() string {
		global.row.lk.Lock()
		defer global.row.lk.Unlock()
		return *paletteName
	}
	waitFor := func(want string) {
		t.Helper()
		for i := 0; i < 200; i++ {
			if current() == want {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatalf("palette %q; want %q", current(), want)
	}
	dark.Store(true)
	waitFor("doom-dark")
	dark.Store(false)
	waitFor("doom-light")

	// With auto off, the appearance is not followed.
	global.row.lk.Lock()
	autoPalette.on = false
	global.row.lk.Unlock()
	dark.Store(true)
	time.Sleep(60 * time.Millisecond)
	if p := current(); p != "doom-light" {
		t.Errorf("auto off, yet the palette followed the system: %q", p)
	}
}
