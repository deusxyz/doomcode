package theme

import (
	"sort"
	"strings"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/frame"
)

// A StyleSpec says how a named text style is painted on top of a
// palette's text colours. A zero Fg or Bg inherits the frame's text or
// background colour. Underline draws a line in Line, or in Fg (or the
// text colour) when Line is zero.
//
// Style names come from external programs through the window's style
// file; see docs/05-style-spec.md section 4 in the justcode project.
type StyleSpec struct {
	Fg, Bg    ColorSpec
	Underline bool
	Line      ColorSpec
}

// Styles maps style names to specs for one palette. Names are dotted:
// "keyword.control" falls back to "keyword" when it has no entry.
type Styles map[string]StyleSpec

// Resolve looks name up, trying shorter dotted prefixes. ok is false
// when nothing matches; such a style is drawn as plain text.
func (st Styles) Resolve(name string) (StyleSpec, bool) {
	for {
		if s, ok := st[name]; ok {
			return s, true
		}
		i := strings.LastIndexByte(name, '.')
		if i < 0 {
			return StyleSpec{}, false
		}
		name = name[:i]
	}
}

// Names returns the defined style names, sorted.
func (st Styles) Names() []string {
	names := make([]string, 0, len(st))
	for n := range st {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func fg(c draw.Color) StyleSpec { return StyleSpec{Fg: solid(c)} }
func bgLine(bg, line draw.Color) StyleSpec {
	return StyleSpec{Bg: solid(bg), Underline: true, Line: solid(line)}
}
func bg(c draw.Color) StyleSpec { return StyleSpec{Bg: solid(c)} }

// Muted colours for the light acme palette: enough to tell things apart
// without turning the pale yellow window into a Christmas tree.
var lightStyles = Styles{
	"comment":     fg(0x6B7F6BFF),
	"keyword":     fg(0x1F3A93FF),
	"string":      fg(0x8B4A1FFF),
	"number":      fg(0x6A2E8FFF),
	"constant":    fg(0x7A3E7AFF),
	"type":        fg(0x1C6E6EFF),
	"function":    fg(0x2F4F8FFF),
	"preproc":     fg(0x7A5C00FF),
	"heading":     fg(0x1F3A93FF),
	"link":        StyleSpec{Fg: solid(0x1A5FB4FF), Underline: true},
	"error":       bgLine(0xFFD6D6FF, 0xC02020FF),
	"warning":     bgLine(0xFFF3C4FF, 0xB07A00FF),
	"info":        bgLine(0xDDEBFFFF, 0x2A5FB4FF),
	"hint":        bgLine(0xE6F2E6FF, 0x3A8A3AFF),
	"match":       bg(0xFFE28AFF),
	"diff.add":    bg(0xDDF5DDFF),
	"diff.del":    bg(0xFFDDDDFF),
	"diff.change": bg(0xFFF0C8FF),
}

// Brighter colours for the dark vampira palette (text on 0x222222).
var darkStyles = Styles{
	"comment":     fg(0x8A9A8AFF),
	"keyword":     fg(0x82AAFFFF),
	"string":      fg(0xE0A070FF),
	"number":      fg(0xC792EAFF),
	"constant":    fg(0xD0A0E0FF),
	"type":        fg(0x80CBC4FF),
	"function":    fg(0x89B4FAFF),
	"preproc":     fg(0xD7BA7DFF),
	"heading":     fg(0x82AAFFFF),
	"link":        StyleSpec{Fg: solid(0x7FB4FFFF), Underline: true},
	"error":       bgLine(0x5A2A2AFF, 0xFF6B6BFF),
	"warning":     bgLine(0x5A4A1AFF, 0xFFD166FF),
	"info":        bgLine(0x203A5AFF, 0x7FB4FFFF),
	"hint":        bgLine(0x2A4A2AFF, 0x8BD48BFF),
	"match":       bg(0x6A5A1AFF),
	"diff.add":    bg(0x1F4A2AFF),
	"diff.del":    bg(0x5A2A2AFF),
	"diff.change": bg(0x4A4020FF),
}

// Solarized accents, shared by both solarized palettes; only the
// diagnostic backgrounds differ between light and dark.
const (
	solCyan    draw.Color = 0x2aa198FF
	solOrange  draw.Color = 0xcb4b16FF
	solMagenta draw.Color = 0xd33682FF
)

func solarizedStyles(dark bool) Styles {
	comment := solBase1
	errBg, warnBg, infoBg, hintBg, matchBg := draw.Color(0xF5D9D5FF), draw.Color(0xF3E6C3FF), draw.Color(0xD9E6F2FF), draw.Color(0xDCEBD0FF), draw.Color(0xEBDDAAFF)
	addBg, delBg, chgBg := draw.Color(0xDCEBD0FF), draw.Color(0xF5D9D5FF), draw.Color(0xF3E6C3FF)
	if dark {
		comment = solBase01
		errBg, warnBg, infoBg, hintBg, matchBg = 0x3A1E24FF, 0x3A3320FF, 0x14384AFF, 0x1E3A2AFF, 0x3A3A14FF
		addBg, delBg, chgBg = 0x1E3A2AFF, 0x3A1E24FF, 0x3A3320FF
	}
	return Styles{
		"comment":     fg(comment),
		"keyword":     fg(solGreen),
		"string":      fg(solCyan),
		"number":      fg(solMagenta),
		"constant":    fg(solMagenta),
		"type":        fg(solYellow),
		"function":    fg(solBlue),
		"preproc":     fg(solOrange),
		"heading":     fg(solBlue),
		"link":        StyleSpec{Fg: solid(solBlue), Underline: true},
		"error":       bgLine(errBg, solRed),
		"warning":     bgLine(warnBg, solYellow),
		"info":        bgLine(infoBg, solBlue),
		"hint":        bgLine(hintBg, solGreen),
		"match":       bg(matchBg),
		"diff.add":    bg(addBg),
		"diff.del":    bg(delBg),
		"diff.change": bg(chgBg),
	}
}

// StyleSet turns style names into frame style indices for one display.
// Index 0 is plain text. Names are given an index on first use, whether
// or not the palette knows them: an unknown name is drawn as plain text
// but keeps its index, so a theme that learns it later needs no rewrite.
type StyleSet struct {
	display draw.Display
	styles  Styles
	names   []string // index -> name; names[0] is ""
	index   map[string]uint8
	cols    []frame.StyleColours
}

// NewStyleSet returns an empty set that paints with st on display.
func NewStyleSet(display draw.Display, st Styles) *StyleSet {
	return &StyleSet{
		display: display,
		styles:  st,
		names:   []string{""},
		index:   map[string]uint8{"": 0},
		cols:    []frame.StyleColours{{}},
	}
}

// Index returns the style index for name, allocating one on first use.
// With all 255 indices taken, further names are drawn as plain text.
func (ss *StyleSet) Index(name string) uint8 {
	if i, ok := ss.index[name]; ok {
		return i
	}
	if len(ss.names) >= 256 {
		return 0
	}
	i := uint8(len(ss.names))
	ss.names = append(ss.names, name)
	ss.index[name] = i
	ss.cols = append(ss.cols, ss.colours(name))
	return i
}

// Name is the inverse of Index.
func (ss *StyleSet) Name(i uint8) string {
	if int(i) < len(ss.names) {
		return ss.names[i]
	}
	return ""
}

// Len is the number of allocated indices, including 0.
func (ss *StyleSet) Len() int { return len(ss.names) }

// Table is the frame style table for the current indices. The slice
// grows as names are allocated, so frames compare its length with the
// one they installed last.
func (ss *StyleSet) Table() []frame.StyleColours { return ss.cols }

func (ss *StyleSet) colours(name string) frame.StyleColours {
	spec, ok := ss.styles.Resolve(name)
	if !ok || ss.display == nil {
		return frame.StyleColours{}
	}
	var c frame.StyleColours
	if spec.Fg.Color != 0 {
		c.Text = AllocOne(ss.display, spec.Fg)
	}
	if spec.Bg.Color != 0 {
		c.Back = AllocOne(ss.display, spec.Bg)
	}
	c.Underline = spec.Underline
	if spec.Line.Color != 0 {
		c.Line = AllocOne(ss.display, spec.Line)
	}
	return c
}
