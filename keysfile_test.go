package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rjkroege/edwood/draw"
)

func TestLoadKeysText(t *testing.T) {
	km := DefaultKeymap()
	n, errs := km.LoadKeysText(strings.NewReader(`
# comment line
C-q    put
F2     execute
C-k    -          # unbind kill-line
Home   file-start
bogus line here
C-s
`))
	if n != 3 {
		t.Errorf("applied %d lines; want 3", n)
	}
	if len(errs) != 3 {
		t.Fatalf("got %d errors; want 3: %v", len(errs), errs)
	}
	for i, want := range []string{"line 6: unknown action", "line 7:", "line 8:"} {
		if !strings.Contains(errs[i].Error(), want) {
			t.Errorf("errs[%d] = %v; want it to mention %q", i, errs[i], want)
		}
	}
	if a := km.Lookup(0x11); a == nil || a.Name != "put" {
		t.Errorf("C-q = %v; want put", a)
	}
	if a := km.Lookup(KF | 2); a == nil || a.Name != "execute" {
		t.Errorf("F2 = %v; want execute", a)
	}
	if a := km.Lookup(0x0b); a != nil {
		t.Errorf("C-k still bound to %s after unbind", a.Name)
	}
	// A failed line leaves the default in place.
	if a := km.Lookup(draw.KeyHome); a == nil || a.Name != "line-start" {
		t.Errorf("Home = %v; want line-start (default kept after bad line)", a)
	}
}

func TestLoadKeysFile(t *testing.T) {
	defer func() { global.keymap = DefaultKeymap() }()

	dir := t.TempDir()
	path := filepath.Join(dir, "keys")

	// Missing file: defaults, no error.
	n, errs := loadKeysFile(path)
	if n != 0 || len(errs) != 0 {
		t.Errorf("missing file: n=%d errs=%v; want 0, none", n, errs)
	}
	if a := global.keymap.Lookup(0x05); a == nil || a.Name != "execute" {
		t.Errorf("defaults not installed: C-e = %v", a)
	}

	if err := os.WriteFile(path, []byte("C-e look\nC-o execute\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	n, errs = loadKeysFile(path)
	if n != 2 || len(errs) != 0 {
		t.Errorf("n=%d errs=%v; want 2, none", n, errs)
	}
	if a := global.keymap.Lookup(0x05); a == nil || a.Name != "look" {
		t.Errorf("C-e = %v; want look", a)
	}
	if a := global.keymap.Lookup(0x0f); a == nil || a.Name != "execute" {
		t.Errorf("C-o = %v; want execute", a)
	}
	// Reloading starts from the defaults again, not from the previous map.
	if err := os.WriteFile(path, []byte("F5 put\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	loadKeysFile(path)
	if a := global.keymap.Lookup(0x05); a == nil || a.Name != "execute" {
		t.Errorf("after reload C-e = %v; want execute (default restored)", a)
	}
	if a := global.keymap.Lookup(KF | 5); a == nil || a.Name != "put" {
		t.Errorf("after reload F5 = %v; want put", a)
	}
}

func TestKeysFilePath(t *testing.T) {
	t.Setenv("EDWOOD_KEYS", "/tmp/x/keys")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	t.Setenv("HOME", "/tmp/home")
	if got := keysFilePath(); got != "/tmp/x/keys" {
		t.Errorf("EDWOOD_KEYS: got %q", got)
	}
	t.Setenv("EDWOOD_KEYS", "")
	if got, want := keysFilePath(), filepath.Join("/tmp/xdg", "edwood", "keys"); got != want {
		t.Errorf("XDG_CONFIG_HOME: got %q; want %q", got, want)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	if got, want := keysFilePath(), filepath.Join("/tmp/home", ".config", "edwood", "keys"); got != want {
		t.Errorf("HOME: got %q; want %q", got, want)
	}
}

func TestActionsDoc(t *testing.T) {
	doc := actionsDoc()
	for _, name := range []string{"execute", "look", "put", "cursor-up"} {
		if !strings.Contains(doc, name) {
			t.Errorf("actionsDoc lacks %q", name)
		}
	}
	lines := strings.Split(strings.TrimSpace(doc), "\n")
	if len(lines) != len(actionTable) {
		t.Errorf("actionsDoc has %d lines; want %d", len(lines), len(actionTable))
	}
}
