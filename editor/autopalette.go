package main

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// palette auto: the light or dark palette following the system's
// appearance. devdraw does not report changes, so the appearance is
// polled every five seconds (the user's choice: cheap enough not to
// burden the system).
//
// See docs/09-config-spec.md section 5.1 in the doomcode repository.

var autoPalettePoll = 5 * time.Second // a variable for tests

// autoPalette is on while the palette follows the system; "Theme NAME"
// turns it off, "Theme auto" back on. Guarded by the row lock once the
// editor runs.
var autoPalette struct {
	on          bool
	light, dark string
}

// systemDark reports whether the system uses a dark appearance. Errors
// count as light: macOS's "defaults read" fails in light mode.
func systemDark() bool {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output()
		return err == nil && strings.Contains(string(out), "Dark")
	case "linux", "freebsd", "openbsd", "netbsd":
		out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme").Output()
		return err == nil && strings.Contains(string(out), "dark")
	}
	return false
}

// autoPaletteName is the palette for the given appearance.
func autoPaletteName(dark bool) string {
	if dark {
		return autoPalette.dark
	}
	return autoPalette.light
}

// autoPaletteLoop switches the palette when the appearance changes.
func autoPaletteLoop(ctx context.Context, g *globals, dark func() bool) {
	t := time.NewTicker(autoPalettePoll)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		// autoPalette is changed by commands, which hold the row lock;
		// ask the system without holding it.
		g.row.lk.Lock()
		on := autoPalette.on
		g.row.lk.Unlock()
		if !on {
			continue
		}
		isDark := dark()
		g.row.lk.Lock()
		want := autoPaletteName(isDark)
		if autoPalette.on && want != *paletteName {
			g.switchPalette(want)
		}
		g.row.lk.Unlock()
	}
}
