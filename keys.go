package main

import (
	"fmt"
	"image"
	"sort"
	"strconv"
	"strings"

	"github.com/rjkroege/edwood/draw"
)

// An Action is a named editing command that a key can be bound to.
// Actions operate on the Text the keystroke was delivered to, with the
// window already locked by the caller, exactly like the mouse paths.
type Action struct {
	Name string
	Doc  string
	// Mutates reports whether the action may change the buffer, in which
	// case Text.Type establishes an undo point before running it.
	Mutates bool
	// Row marks an action on windows and columns rather than on a text:
	// it tolerates a nil focus text. Text actions are skipped when there
	// is no focus.
	Row bool
	Fn  func(t *Text)
}

// Keymap maps a keyboard rune, as delivered by devdraw, to an Action.
type Keymap map[rune]*Action

// Lookup returns the action bound to r, or nil if r is ordinary typing.
func (km Keymap) Lookup(r rune) *Action {
	if km == nil {
		return nil
	}
	return km[r]
}

// Bind binds the key named by key (see ParseKey) to the named action.
func (km Keymap) Bind(key, action string) error {
	r, err := ParseKey(key)
	if err != nil {
		return err
	}
	a, ok := actionTable[action]
	if !ok {
		return fmt.Errorf("unknown action %q", action)
	}
	km[r] = a
	return nil
}

// Unbind removes the binding for key so that it is typed as text again.
func (km Keymap) Unbind(key string) error {
	r, err := ParseKey(key)
	if err != nil {
		return err
	}
	delete(km, r)
	return nil
}

// String lists the bindings, one per line, sorted by key name. It is the
// format shown by the Keys command.
func (km Keymap) String() string {
	type binding struct{ key, action string }
	var bs []binding
	for r, a := range km {
		bs = append(bs, binding{KeyName(r), a.Name})
	}
	sort.Slice(bs, func(i, j int) bool { return bs[i].key < bs[j].key })
	var sb strings.Builder
	for _, b := range bs {
		fmt.Fprintf(&sb, "%-12s %s\n", b.key, b.action)
	}
	return sb.String()
}

// Named special keys, in the spelling used by key names and the Keys file.
var specialKeys = map[string]rune{
	"Left":      draw.KeyLeft,
	"Right":     draw.KeyRight,
	"Up":        draw.KeyUp,
	"Down":      draw.KeyDown,
	"Home":      draw.KeyHome,
	"End":       draw.KeyEnd,
	"PgUp":      draw.KeyPageUp,
	"PgDn":      draw.KeyPageDown,
	"Ins":       draw.KeyInsert,
	"Del":       0x7F,
	"Backspace": 0x08,
	"Tab":       '\t',
	"Enter":     '\n',
	"Esc":       0x1B,
	"Space":     ' ',
}

var specialKeyNames = func() map[rune]string {
	m := make(map[rune]string, len(specialKeys))
	for name, r := range specialKeys {
		m[r] = name
	}
	return m
}()

// ParseKey converts a key name into the rune devdraw delivers for it.
//
// Forms: a single printable character ("x", "%"); "C-x" for Control plus a
// letter or one of "[ \ ] ^ _"; "Cmd-x" for Command (macOS) plus a
// printable character, "Cmd-X" with a capital for Command-Shift; a special
// key name from specialKeys ("Left", "PgUp", "Enter"); "F1".."F12"; or a
// raw code "0xF800".
func ParseKey(name string) (rune, error) {
	if name == "" {
		return 0, fmt.Errorf("empty key name")
	}
	if r, ok := specialKeys[name]; ok {
		return r, nil
	}
	if strings.HasPrefix(name, "0x") {
		n, err := strconv.ParseUint(name[2:], 16, 32)
		if err != nil {
			return 0, fmt.Errorf("bad key code %q", name)
		}
		return rune(n), nil
	}
	if len(name) >= 2 && name[0] == 'F' {
		if n, err := strconv.Atoi(name[1:]); err == nil && 1 <= n && n <= 12 {
			return KF | rune(n), nil
		}
	}
	if strings.HasPrefix(name, "C-") {
		rest := []rune(name[2:])
		if len(rest) != 1 {
			return 0, fmt.Errorf("bad control key %q", name)
		}
		c := rest[0]
		switch {
		case 'a' <= c && c <= 'z':
			return c - 'a' + 1, nil
		case 'A' <= c && c <= 'Z':
			return c - 'A' + 1, nil
		case c == '[', c == '\\', c == ']', c == '^', c == '_':
			return c - 0x40, nil
		}
		return 0, fmt.Errorf("bad control key %q", name)
	}
	if strings.HasPrefix(name, "Cmd-") {
		rest := []rune(name[4:])
		if len(rest) != 1 || rest[0] < ' ' || rest[0] > '~' {
			return 0, fmt.Errorf("bad command key %q", name)
		}
		return draw.KeyCmd + rest[0], nil
	}
	rest := []rune(name)
	if len(rest) == 1 && rest[0] > ' ' {
		return rest[0], nil
	}
	return 0, fmt.Errorf("unknown key %q", name)
}

// KeyName is the inverse of ParseKey.
func KeyName(r rune) string {
	if name, ok := specialKeyNames[r]; ok {
		return name
	}
	switch {
	case 1 <= r && r <= 26:
		return "C-" + string(r+'a'-1)
	case 0x1c <= r && r <= 0x1f:
		return "C-" + string(r+0x40)
	case draw.KeyCmd+' ' <= r && r <= draw.KeyCmd+'~':
		return "Cmd-" + string(r-draw.KeyCmd)
	case KF|1 <= r && r <= KF|12:
		return "F" + strconv.Itoa(int(r&^KF))
	case r > ' ' && r < 0x7f:
		return string(r)
	}
	return fmt.Sprintf("0x%X", r)
}

// defaultBindings is the built-in keymap: CUA editing keys plus keyboard
// equivalents of the Acme mouse buttons. Cmd- variants are macOS aliases;
// they are harmless elsewhere because devdraw never produces them there.
// See docs/03-keyboard-spec.md in the justcode project.
var defaultBindings = []struct{ key, action string }{
	{"Left", "cursor-left"},
	{"Right", "cursor-right"},
	{"Up", "cursor-up"},
	{"Down", "cursor-down"},
	{"0xF800", "cursor-down"}, // some devdraws send this for Down
	{"PgUp", "page-up"},
	{"PgDn", "page-down"},
	{"Home", "line-start"},
	{"End", "line-end"},

	{"C-a", "select-all"},
	{"Cmd-a", "select-all"},
	{"C-c", "snarf"},
	{"Cmd-c", "snarf"},
	{"C-x", "cut"},
	{"Cmd-x", "cut"},
	{"C-v", "paste"},
	{"Cmd-v", "paste"},
	{"C-z", "undo"},
	{"Cmd-z", "undo"},
	{"C-y", "redo"},
	{"Cmd-Z", "redo"},
	{"C-k", "kill-line"},

	{"C-e", "execute"},
	{"Cmd-e", "execute"},
	{"C-o", "look"},
	{"Cmd-o", "look"},
	{"C-s", "put"},
	{"Cmd-s", "put"},
}

// DefaultKeymap returns a fresh copy of the built-in bindings.
func DefaultKeymap() Keymap {
	km := make(Keymap, len(defaultBindings))
	for _, b := range defaultBindings {
		if err := km.Bind(b.key, b.action); err != nil {
			panic("default keymap: " + err.Error())
		}
	}
	return km
}

// actionTable is every action a key may be bound to, by name. It is a
// package-level variable rather than filled in by init so that it exists
// before globals.go's init builds the default keymap.
var actionTable = buildActionTable()

func buildActionTable() map[string]*Action {
	table := make(map[string]*Action)
	for _, a := range []*Action{
		{Name: "cursor-left", Doc: "move left one character; collapse a selection to its start", Fn: (*Text).keyCursorLeft},
		{Name: "cursor-right", Doc: "move right one character; collapse a selection to its end", Fn: (*Text).keyCursorRight},
		{Name: "cursor-up", Doc: "move up one line keeping the column; in a tag, shrink it", Fn: (*Text).keyCursorUp},
		{Name: "cursor-down", Doc: "move down one line keeping the column; in a tag, expand it", Fn: (*Text).keyCursorDown},
		{Name: "page-up", Doc: "scroll up two thirds of a screen", Fn: (*Text).keyPageUp},
		{Name: "page-down", Doc: "scroll down two thirds of a screen", Fn: (*Text).keyPageDown},
		{Name: "line-start", Doc: "move to the beginning of the line", Fn: (*Text).keyLineStart},
		{Name: "line-end", Doc: "move to the end of the line", Fn: (*Text).keyLineEnd},
		{Name: "select-all", Doc: "select the whole text", Fn: (*Text).keySelectAll},

		{Name: "snarf", Doc: "copy the selection to the snarf buffer (Snarf)", Fn: (*Text).keySnarf},
		{Name: "cut", Doc: "cut the selection to the snarf buffer (Cut)", Mutates: true, Fn: (*Text).keyCut},
		{Name: "paste", Doc: "replace the selection with the snarf buffer (Paste)", Mutates: true, Fn: (*Text).keyPaste},
		{Name: "undo", Doc: "Undo", Fn: (*Text).keyUndo},
		{Name: "redo", Doc: "Redo", Fn: (*Text).keyRedo},
		{Name: "kill-line", Doc: "delete to the end of the line; on an empty line delete the newline", Mutates: true, Fn: (*Text).keyKillLine},

		{Name: "execute", Doc: "run the selection or the word under the cursor, like button 2", Fn: (*Text).keyExecute},
		{Name: "look", Doc: "open or search for the selection or the word under the cursor, like button 3", Fn: (*Text).keyLook},
		{Name: "put", Doc: "write the window to its file (Put)", Fn: (*Text).keyPut},
	} {
		if _, dup := table[a.Name]; dup {
			panic("duplicate action " + a.Name)
		}
		table[a.Name] = a
	}
	for _, a := range prefixActions() {
		if _, dup := table[a.Name]; dup {
			panic("duplicate action " + a.Name)
		}
		table[a.Name] = a
	}
	return table
}

// keymap returns the keymap in effect for t.
func (t *Text) keymap() Keymap {
	if global != nil && global.keymap != nil {
		return global.keymap
	}
	return nil
}

// markUndo starts a new undo record for a body text.
func (t *Text) markUndo() {
	if t.what == Body {
		global.seq++
		t.file.Mark(global.seq)
	}
}

// scrollLines scrolls the frame by n lines, down for positive n and up for
// negative n, without moving the selection.
func (t *Text) scrollLines(n int) {
	switch {
	case n > 0:
		q0 := t.org + t.fr.Charofpt(t.fr.Rect().Min.Add(image.Pt(0, n*t.fr.DefaultFontHeight())))
		t.SetOrigin(q0, true)
	case n < 0:
		q0 := t.BackNL(t.org, -n)
		t.SetOrigin(q0, true)
	}
}

func (t *Text) keyCursorLeft() {
	t.TypeCommit()
	if t.q0 > 0 {
		if t.q0 != t.q1 {
			t.Show(t.q0, t.q0, true)
		} else {
			t.Show(t.q0-1, t.q0-1, true)
		}
	}
}

func (t *Text) keyCursorRight() {
	t.TypeCommit()
	if t.q1 < t.file.Nr() {
		// This is a departure from the plan9/plan9port acme
		// Instead of always going right one char from q1, it
		// collapses multi-character selections first, behaving
		// like every other selection on modern systems. -flux
		if t.q0 != t.q1 {
			t.Show(t.q1, t.q1, true)
		} else {
			t.Show(t.q1+1, t.q1+1, true)
		}
	}
}

func (t *Text) keyCursorUp() {
	if t.what == Tag {
		t.tagShrink()
		return
	}
	t.TypeCommit()
	t.moveVertical(-1)
}

func (t *Text) keyCursorDown() {
	if t.what == Tag {
		t.tagExpand()
		return
	}
	t.TypeCommit()
	t.moveVertical(1)
}

func (t *Text) keyPageUp() {
	t.scrollLines(-2 * t.fr.GetFrameFillStatus().Maxlines / 3)
}

func (t *Text) keyPageDown() {
	t.scrollLines(2 * t.fr.GetFrameFillStatus().Maxlines / 3)
}

func (t *Text) keyLineStart() {
	t.TypeCommit()
	q0 := t.lineStart(t.q0)
	t.Show(q0, q0, true)
}

func (t *Text) keyLineEnd() {
	t.TypeCommit()
	q0 := t.lineEnd(t.q1)
	t.Show(q0, q0, true)
}

func (t *Text) keySelectAll() {
	t.TypeCommit()
	t.SetSelect(0, t.file.Nr())
}

func (t *Text) keySnarf() {
	t.TypeCommit()
	cut(t, t, nil, true, false, "")
}

func (t *Text) keyCut() {
	t.markUndo()
	t.TypeCommit()
	t.markUndo()
	cut(t, t, nil, true, true, "")
	t.Show(t.q0, t.q0, true)
	t.iq1 = t.q0
}

func (t *Text) keyPaste() {
	t.markUndo()
	t.TypeCommit()
	t.markUndo()
	paste(t, t, nil, true, false, "")
	t.Show(t.q0, t.q1, true)
	t.iq1 = t.q1
}

func (t *Text) keyUndo() {
	t.TypeCommit()
	undo(t, nil, nil, true, false, "")
}

func (t *Text) keyRedo() {
	t.TypeCommit()
	undo(t, nil, nil, false, false, "")
}

func (t *Text) keyKillLine() {
	t.markUndo()
	t.TypeCommit()
	if t.q0 == t.q1 {
		q1 := t.lineEnd(t.q0)
		if q1 == t.q0 && q1 < t.file.Nr() {
			q1++
		}
		if q1 == t.q0 {
			return
		}
		t.SetSelect(t.q0, q1)
	}
	t.markUndo()
	cut(t, t, nil, false, true, "")
	t.Show(t.q0, t.q0, true)
	t.iq1 = t.q0
}

func (t *Text) keyExecute() {
	t.commitAll()
	execute(t, t.q0, t.q1, false, nil)
}

func (t *Text) keyLook() {
	t.commitAll()
	look3(t, t.q0, t.q1, false)
}

func (t *Text) keyPut() {
	t.commitAll()
	put(t, nil, nil, false, false, "")
}
