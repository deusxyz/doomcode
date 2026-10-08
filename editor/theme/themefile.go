package theme

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/deusxyz/doomcode/editor/draw"
)

// The theme file lets the user adjust colours without rebuilding. It is
// applied on top of a built-in palette, so it only needs to list changes.
//
// Lines (blank lines and # comments ignored):
//
//	palette vampira                  base palette, unless -palette is given
//	style keyword fg=#1f3a93         text style: fg=, bg=, line= colours,
//	style error bg=#ffd6d6 underline   "underline"/"nounderline"
//	style variable -                 remove a style (drawn as plain text)
//	text.back #ffffea                palette slot: tag.back tag.high tag.bord
//	tag.text #000000 mix #ffffff       tag.text tag.htext tag.tick, text.*,
//	                                   ui.modbutton ui.colbutton ui.but2 ui.but3;
//	                                   "mix" gives a 50% blend as the palettes do
//
// Colours are #rgb, #rrggbb or #rrggbbaa.
//
// See docs/05-style-spec.md section 4 in the justcode project.

// A ThemeFile is a parsed theme file.
type ThemeFile struct {
	Palette string                // base palette name, "" if not given
	Slots   map[string]ColorSpec  // palette slot overrides
	Styles  map[string]*StyleSpec // style overrides; nil removes the style
}

// ParseColor parses #rgb, #rrggbb or #rrggbbaa into a draw.Color (RGBA).
func ParseColor(s string) (draw.Color, error) {
	if !strings.HasPrefix(s, "#") {
		return 0, fmt.Errorf("colour %q: want #rrggbb", s)
	}
	h := s[1:]
	switch len(h) {
	case 3:
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]}) + "ff"
	case 6:
		h += "ff"
	case 8:
	default:
		return 0, fmt.Errorf("colour %q: want #rgb, #rrggbb or #rrggbbaa", s)
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("colour %q: %v", s, err)
	}
	return draw.Color(v), nil
}

// ParseTheme reads a theme file. Lines that cannot be applied are
// reported in errs and skipped; the rest are kept.
func ParseTheme(r io.Reader) (ThemeFile, []error) {
	tf := ThemeFile{Slots: map[string]ColorSpec{}, Styles: map[string]*StyleSpec{}}
	var errs []error
	sc := bufio.NewScanner(r)
	for lineno := 1; sc.Scan(); lineno++ {
		line := stripComment(sc.Text())
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		fail := func(format string, args ...interface{}) {
			errs = append(errs, fmt.Errorf("line %d: %s", lineno, fmt.Sprintf(format, args...)))
		}
		switch f[0] {
		case "palette":
			if len(f) != 2 {
				fail("want \"palette name\"")
				continue
			}
			tf.Palette = f[1]
		case "style":
			if len(f) < 3 {
				fail("want \"style name key=value...\" or \"style name -\"")
				continue
			}
			name := f[1]
			if len(f) == 3 && f[2] == "-" {
				tf.Styles[name] = nil
				continue
			}
			spec := &StyleSpec{}
			if old, ok := tf.Styles[name]; ok && old != nil {
				*spec = *old
			}
			ok := true
			for _, kv := range f[2:] {
				switch {
				case kv == "underline":
					spec.Underline = true
				case kv == "nounderline":
					spec.Underline = false
				case strings.HasPrefix(kv, "fg="), strings.HasPrefix(kv, "bg="), strings.HasPrefix(kv, "line="):
					key, val, _ := strings.Cut(kv, "=")
					c, err := ParseColor(val)
					if err != nil {
						fail("%v", err)
						ok = false
						break
					}
					switch key {
					case "fg":
						spec.Fg = solid(c)
					case "bg":
						spec.Bg = solid(c)
					case "line":
						spec.Line = solid(c)
					}
				default:
					fail("unknown style attribute %q", kv)
					ok = false
				}
				if !ok {
					break
				}
			}
			if ok {
				tf.Styles[name] = spec
			}
		default:
			// Palette slot: "slot #colour [mix #colour]".
			if !isSlot(f[0]) {
				fail("unknown directive %q", f[0])
				continue
			}
			if len(f) != 2 && !(len(f) == 4 && f[2] == "mix") {
				fail("want \"%s #colour [mix #colour]\"", f[0])
				continue
			}
			c, err := ParseColor(f[1])
			if err != nil {
				fail("%v", err)
				continue
			}
			spec := solid(c)
			if len(f) == 4 {
				m, err := ParseColor(f[3])
				if err != nil {
					fail("%v", err)
					continue
				}
				spec = mixed(c, m)
			}
			tf.Slots[f[0]] = spec
		}
	}
	if err := sc.Err(); err != nil {
		errs = append(errs, err)
	}
	return tf, errs
}

// stripComment removes a # comment from line. A # followed by a hex digit
// is a colour, not a comment, so "text.back #ffffea  # pale" keeps the
// colour and drops the remark.
func stripComment(line string) string {
	for i := 0; i < len(line); i++ {
		if line[i] != '#' {
			continue
		}
		if i+1 < len(line) && isHex(line[i+1]) && i > 0 && line[i-1] != '#' {
			// looks like a colour: skip over it
			j := i + 1
			for j < len(line) && isHex(line[j]) {
				j++
			}
			i = j - 1
			continue
		}
		return line[:i]
	}
	return line
}

func isHex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

// slotNames are the palette slots a theme file may set.
var slotNames = []string{
	"tag.back", "tag.high", "tag.bord", "tag.text", "tag.htext", "tag.tick",
	"text.back", "text.high", "text.bord", "text.text", "text.htext", "text.tick",
	"ui.modbutton", "ui.colbutton", "ui.but2", "ui.but3",
}

func isSlot(name string) bool {
	for _, s := range slotNames {
		if s == name {
			return true
		}
	}
	return false
}

// SlotNames lists the palette slots a theme file may set, sorted.
func SlotNames() []string {
	out := append([]string(nil), slotNames...)
	sort.Strings(out)
	return out
}

// Apply writes the file's slot and style overrides into p. The base
// palette choice is the caller's business (see Palette field). Styles is
// copied before modification so built-in palettes stay intact.
func (tf ThemeFile) Apply(p *Palette) {
	for slot, c := range tf.Slots {
		target := p.slot(slot)
		if target != nil {
			*target = c
		}
	}
	if len(tf.Styles) == 0 {
		return
	}
	styles := make(Styles, len(p.Styles)+len(tf.Styles))
	for k, v := range p.Styles {
		styles[k] = v
	}
	for name, spec := range tf.Styles {
		if spec == nil {
			delete(styles, name)
			continue
		}
		styles[name] = *spec
	}
	p.Styles = styles
}

// slot returns the ColorSpec a slot name refers to in p.
func (p *Palette) slot(name string) *ColorSpec {
	part, field, ok := strings.Cut(name, ".")
	if !ok {
		return nil
	}
	var fp *FramePalette
	switch part {
	case "tag":
		fp = &p.Tag
	case "text":
		fp = &p.Text
	case "ui":
		switch field {
		case "modbutton":
			return &p.Ui.ModButton
		case "colbutton":
			return &p.Ui.ColButton
		case "but2":
			return &p.Ui.But2
		case "but3":
			return &p.Ui.But3
		}
		return nil
	default:
		return nil
	}
	switch field {
	case "back":
		return &fp.Back
	case "high":
		return &fp.High
	case "bord":
		return &fp.Bord
	case "text":
		return &fp.Text
	case "htext":
		return &fp.HText
	case "tick":
		return &fp.Tick
	}
	return nil
}

// ColorString formats a colour as #rrggbb, or #rrggbbaa when not opaque.
func ColorString(c draw.Color) string {
	if uint32(c)&0xff == 0xff {
		return fmt.Sprintf("#%06x", uint32(c)>>8)
	}
	return fmt.Sprintf("#%08x", uint32(c))
}

// StylesDoc lists the styles of p in theme file syntax, sorted by name.
func (p *Palette) StylesDoc() string {
	var sb strings.Builder
	for _, name := range p.Styles.Names() {
		s := p.Styles[name]
		fmt.Fprintf(&sb, "style %-12s", name)
		if s.Fg.Color != 0 {
			fmt.Fprintf(&sb, " fg=%s", ColorString(s.Fg.Color))
		}
		if s.Bg.Color != 0 {
			fmt.Fprintf(&sb, " bg=%s", ColorString(s.Bg.Color))
		}
		if s.Underline {
			sb.WriteString(" underline")
		}
		if s.Line.Color != 0 {
			fmt.Fprintf(&sb, " line=%s", ColorString(s.Line.Color))
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}
