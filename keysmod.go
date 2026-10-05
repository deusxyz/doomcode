package main

import (
	"strings"
	"unicode"
)

// Special keys with modifiers. devdraw normally reports Shift+Left as
// Left; with DEVDRAW_MODKEYS set (Edwood sets it before starting devdraw)
// a patched devdraw reports KeyMod | mods<<5 | key instead, see
// modkeys.h in the plan9port fork. Unpatched devdraws keep sending the
// plain keys and everything below is simply never triggered.
//
// See docs/03-keyboard-spec.md section 7 in the justcode project.

const (
	KeyMod   = 0xF200
	modShift = 1
	modCtl   = 2
	modAlt   = 4
	modCmd   = 8
)

// modKeyNames index the special keys the protocol can modify; index 0 is
// unused. The names are the key names ParseKey accepts.
var modKeyNames = []string{"", "Left", "Right", "Up", "Down", "Home", "End", "PgUp", "PgDn", "Ins", "Del", "Backspace", "Tab", "Enter", "Esc"}

func modKey(mods int, index int) rune { return rune(KeyMod | mods<<5 | index) }

// modKeyIndex returns the modkey index of a special key name, or 0.
func modKeyIndex(name string) int {
	for i, n := range modKeyNames {
		if i > 0 && n == name {
			return i
		}
	}
	return 0
}

// parseModKey parses names like "S-Left", "C-S-Right", "M-Enter",
// "Cmd-Up": any of the prefixes S-, C-, M-, Cmd- followed by a special
// key. ok is false when name is not of that form.
func parseModKey(name string) (rune, bool) {
	mods := 0
	rest := name
	for {
		switch {
		case strings.HasPrefix(rest, "Cmd-"):
			mods |= modCmd
			rest = rest[4:]
		case strings.HasPrefix(rest, "S-"):
			mods |= modShift
			rest = rest[2:]
		case strings.HasPrefix(rest, "C-"):
			mods |= modCtl
			rest = rest[2:]
		case strings.HasPrefix(rest, "M-"):
			mods |= modAlt
			rest = rest[2:]
		default:
			if mods == 0 {
				return 0, false
			}
			i := modKeyIndex(rest)
			if i == 0 {
				return 0, false
			}
			return modKey(mods, i), true
		}
	}
}

// modKeyName is the inverse of parseModKey for runes in the KeyMod range.
func modKeyName(r rune) (string, bool) {
	if r < KeyMod || r > KeyMod|(15<<5)|31 {
		return "", false
	}
	mods := int(r>>5) & 15
	i := int(r) & 31
	if i <= 0 || i >= len(modKeyNames) || mods == 0 {
		return "", false
	}
	var sb strings.Builder
	if mods&modCtl != 0 {
		sb.WriteString("C-")
	}
	if mods&modShift != 0 {
		sb.WriteString("S-")
	}
	if mods&modAlt != 0 {
		sb.WriteString("M-")
	}
	if mods&modCmd != 0 {
		sb.WriteString("Cmd-")
	}
	sb.WriteString(modKeyNames[i])
	return sb.String(), true
}

// modifiedBindings are the defaults for keys that need the patched
// devdraw. Shift extends the selection through the anchor; Ctrl moves by
// words; Cmd follows macOS habits.
var modifiedBindings = []struct{ key, action string }{
	{"S-Left", "select-left"},
	{"S-Right", "select-right"},
	{"S-Up", "select-up"},
	{"S-Down", "select-down"},
	{"S-Home", "select-line-start"},
	{"S-End", "select-line-end"},
	{"C-Left", "word-left"},
	{"C-Right", "word-right"},
	{"C-S-Left", "select-word-left"},
	{"C-S-Right", "select-word-right"},
	{"C-Home", "file-start"},
	{"C-End", "file-end"},
	{"C-Up", "scroll-up"},
	{"C-Down", "scroll-down"},
	{"C-Enter", "execute"},
	{"Cmd-Enter", "execute"},
	{"M-Enter", "look"},
	{"C-Backspace", "delete-word-back"},
	{"C-Del", "delete-word-forward"},
	{"S-Tab", "outdent"},
	{"Cmd-Left", "line-start"},
	{"Cmd-Right", "line-end"},
	{"Cmd-Up", "file-start"},
	{"Cmd-Down", "file-end"},
	{"Cmd-S-Left", "select-line-start"},
	{"Cmd-S-Right", "select-line-end"},
}

// modifiedActions are the actions only these keys use.
func modifiedActions() []*Action {
	sel := func(name, doc string, move func(*Text)) *Action {
		return &Action{Name: name, Doc: doc, Fn: func(t *Text) {
			if !t.anchorOn {
				t.keyAnchor()
			}
			move(t)
		}}
	}
	return []*Action{
		sel("select-left", "extend the selection one character left", (*Text).keyCursorLeft),
		sel("select-right", "extend the selection one character right", (*Text).keyCursorRight),
		sel("select-up", "extend the selection one line up", (*Text).keyCursorUp),
		sel("select-down", "extend the selection one line down", (*Text).keyCursorDown),
		sel("select-line-start", "extend the selection to the start of the line", (*Text).keyLineStart),
		sel("select-line-end", "extend the selection to the end of the line", (*Text).keyLineEnd),
		sel("select-word-left", "extend the selection one word left", (*Text).keyWordLeft),
		sel("select-word-right", "extend the selection one word right", (*Text).keyWordRight),
		{Name: "word-left", Doc: "move to the start of the previous word", Fn: (*Text).keyWordLeft},
		{Name: "word-right", Doc: "move to the end of the next word", Fn: (*Text).keyWordRight},
		{Name: "scroll-up", Doc: "scroll one line up without moving the cursor", Fn: func(t *Text) { t.scrollLines(-1) }},
		{Name: "scroll-down", Doc: "scroll one line down without moving the cursor", Fn: func(t *Text) { t.scrollLines(1) }},
		{Name: "delete-word-back", Doc: "delete the word before the cursor", Mutates: true, Fn: (*Text).keyDeleteWordBack},
		{Name: "delete-word-forward", Doc: "delete the word after the cursor", Mutates: true, Fn: (*Text).keyDeleteWordForward},
	}
}

func isWordRune(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }

// wordLeft returns the start of the word before q: spaces are skipped,
// then letters, digits and underscores; a run of other punctuation counts
// as a word too.
func (t *Text) wordLeft(q int) int {
	for q > 0 && unicode.IsSpace(t.file.ReadC(q-1)) && t.file.ReadC(q-1) != '\n' {
		q--
	}
	if q > 0 && t.file.ReadC(q-1) == '\n' {
		return q - 1
	}
	if q == 0 {
		return 0
	}
	word := isWordRune(t.file.ReadC(q - 1))
	for q > 0 {
		c := t.file.ReadC(q - 1)
		if unicode.IsSpace(c) || isWordRune(c) != word {
			break
		}
		q--
	}
	return q
}

// wordRight returns the end of the word after q, mirroring wordLeft.
func (t *Text) wordRight(q int) int {
	n := t.file.Nr()
	for q < n && unicode.IsSpace(t.file.ReadC(q)) && t.file.ReadC(q) != '\n' {
		q++
	}
	if q < n && t.file.ReadC(q) == '\n' {
		return q + 1
	}
	if q >= n {
		return n
	}
	word := isWordRune(t.file.ReadC(q))
	for q < n {
		c := t.file.ReadC(q)
		if unicode.IsSpace(c) || isWordRune(c) != word {
			break
		}
		q++
	}
	return q
}

func (t *Text) keyWordLeft() {
	t.TypeCommit()
	q := t.q0
	if t.anchorOn {
		q = t.caret()
	}
	t.moveCaret(t.wordLeft(q))
}

func (t *Text) keyWordRight() {
	t.TypeCommit()
	q := t.q1
	if t.anchorOn {
		q = t.caret()
	}
	t.moveCaret(t.wordRight(q))
}

func (t *Text) keyDeleteWordBack() {
	t.dropAnchor()
	t.markUndo()
	t.TypeCommit()
	if t.q0 != t.q1 {
		cut(t, t, nil, false, true, "")
		return
	}
	q0 := t.wordLeft(t.q0)
	if q0 == t.q0 {
		return
	}
	t.markUndo()
	t.Delete(q0, t.q0, true)
	t.Show(q0, q0, true)
	t.iq1 = q0
}

func (t *Text) keyDeleteWordForward() {
	t.dropAnchor()
	t.markUndo()
	t.TypeCommit()
	if t.q0 != t.q1 {
		cut(t, t, nil, false, true, "")
		return
	}
	q1 := t.wordRight(t.q0)
	if q1 == t.q0 {
		return
	}
	t.markUndo()
	t.Delete(t.q0, q1, true)
	t.Show(t.q0, t.q0, true)
	t.iq1 = t.q0
}
