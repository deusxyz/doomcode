package frame

import (
	"github.com/rjkroege/edwood/draw"
)

// StyleColours is how a style index is painted. A nil Text or Back means
// the frame's own ColText or ColBack. Index 0 of a style table is ignored:
// style 0 is always plain text.
type StyleColours struct {
	Text      draw.Image
	Back      draw.Image
	Underline bool
	Line      draw.Image // underline colour; nil means the text colour
}

// SetStyleTable installs the colours for style indices; see StyleColours.
// It does not repaint: callers Restyle or redraw the text afterwards.
func (f *frameimpl) SetStyleTable(t []StyleColours) {
	f.lk.Lock()
	defer f.lk.Unlock()
	f.styles = t
}

func (f *frameimpl) style(b *frbox) *StyleColours {
	if b == nil || b.Style == 0 || int(b.Style) >= len(f.styles) {
		return nil
	}
	return &f.styles[b.Style]
}

// styleText is the text colour of b.
func (f *frameimpl) styleText(b *frbox) draw.Image {
	if s := f.style(b); s != nil && s.Text != nil {
		return s.Text
	}
	return f.cols[ColText]
}

// styleBack is the background colour of b.
func (f *frameimpl) styleBack(b *frbox) draw.Image {
	if s := f.style(b); s != nil && s.Back != nil {
		return s.Back
	}
	return f.cols[ColBack]
}

// styleUnderline reports whether b is underlined.
func (f *frameimpl) styleUnderline(b *frbox) bool {
	s := f.style(b)
	return s != nil && s.Underline
}

// styleLine is the underline colour of b.
func (f *frameimpl) styleLine(b *frbox) draw.Image {
	if s := f.style(b); s != nil && s.Line != nil {
		return s.Line
	}
	return f.styleText(b)
}

// Restyle gives the runes in [p0,p1) the styles in styles (one per rune;
// a short slice is padded with style 0) and repaints that range. The text
// and the selection are unchanged.
func (f *frameimpl) Restyle(p0, p1 int, styles []uint8) {
	f.lk.Lock()
	defer f.lk.Unlock()
	f.restyleimpl(p0, p1, styles)
}

func (f *frameimpl) restyleimpl(p0, p1 int, styles []uint8) {
	if p1 > f.nchars {
		p1 = f.nchars
	}
	if p0 < 0 {
		p0 = 0
	}
	if p0 >= p1 || f.background == nil {
		return
	}
	styleAt := func(p int) uint8 {
		if i := p - p0; i < len(styles) {
			return styles[i]
		}
		return 0
	}

	// Put box boundaries at p0 and p1, then at every style change inside.
	n0 := f.findbox(0, 0, p0)
	n1 := f.findbox(n0, p0, p1)
	changed := false
	p := p0
	for nb := n0; nb < n1; nb++ {
		b := f.box[nb]
		nr := nrune(b)
		st := styleAt(p)
		if b.Nrune > 0 {
			// Split where the style changes within the box.
			for i := 1; i < nr; i++ {
				if styleAt(p+i) != st {
					f.splitbox(nb, i)
					n1++
					b = f.box[nb]
					nr = i
					break
				}
			}
		}
		if b.Style != st {
			b.Style = st
			changed = true
		}
		p += nr
	}
	if !changed {
		return
	}

	// Repaint [p0,p1) in the new styles, then put a highlighted selection
	// and the tick back on top.
	wasTicked := f.ticked
	if wasTicked {
		f.tick(f.ptofcharptb(f.sp0, f.rect.Min, 0), false)
	}
	f.drawsel0(f.ptofcharptb(p0, f.rect.Min, 0), p0, p1, nil, nil)
	if f.highlighton && f.sp0 < f.sp1 {
		h0, h1 := f.sp0, f.sp1
		if h0 < p0 {
			h0 = p0
		}
		if h1 > p1 {
			h1 = p1
		}
		if h0 < h1 {
			f.drawsel0(f.ptofcharptb(h0, f.rect.Min, 0), h0, h1, f.cols[ColHigh], f.cols[ColHText])
		}
	}
	if wasTicked {
		f.tick(f.ptofcharptb(f.sp0, f.rect.Min, 0), true)
	}
}
