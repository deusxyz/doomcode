package file

import (
	"sort"
)

// A Span is a run of text [Q0,Q1) in rune offsets that carries a named
// style ("keyword", "comment", "error", ...). What a name looks like is
// the theme's business; the table only stores names.
type Span struct {
	Q0, Q1 int
	Name   string
}

// StyleTable holds the styled spans of one buffer: sorted by Q0, never
// overlapping, never empty. It belongs to the ObservableEditableBuffer,
// which keeps it in step with insertions and deletions, so windows that
// share a buffer (Zerox) share its styles.
//
// See docs/05-style-spec.md in the justcode project.
type StyleTable struct {
	spans []Span
	// version counts changes, including shifts, so that observers can
	// tell whether anything happened since they last looked.
	version int
}

// Version increments on every change to the table.
func (st *StyleTable) Version() int { return st.version }

// Len is the number of spans.
func (st *StyleTable) Len() int { return len(st.spans) }

// Spans returns a copy of all spans.
func (st *StyleTable) Spans() []Span {
	return append([]Span(nil), st.spans...)
}

// find returns the index of the first span with Q1 > q, i.e. the first
// span that could contain or follow q.
func (st *StyleTable) find(q int) int {
	return sort.Search(len(st.spans), func(i int) bool { return st.spans[i].Q1 > q })
}

// At returns the span containing q.
func (st *StyleTable) At(q int) (Span, bool) {
	i := st.find(q)
	if i < len(st.spans) && st.spans[i].Q0 <= q {
		return st.spans[i], true
	}
	return Span{}, false
}

// Range returns the spans intersecting [q0,q1), clipped to it.
func (st *StyleTable) Range(q0, q1 int) []Span {
	var out []Span
	for i := st.find(q0); i < len(st.spans) && st.spans[i].Q0 < q1; i++ {
		s := st.spans[i]
		if s.Q0 < q0 {
			s.Q0 = q0
		}
		if s.Q1 > q1 {
			s.Q1 = q1
		}
		if s.Q0 < s.Q1 {
			out = append(out, s)
		}
	}
	return out
}

// Set gives [q0,q1) the style name, replacing whatever was there.
// Adjacent spans with the same name are merged.
func (st *StyleTable) Set(q0, q1 int, name string) {
	if q0 >= q1 || name == "" {
		return
	}
	st.cut(q0, q1, nil)
	i := st.find(q0)
	st.spans = append(st.spans, Span{})
	copy(st.spans[i+1:], st.spans[i:])
	st.spans[i] = Span{Q0: q0, Q1: q1, Name: name}
	st.mergeAround(i)
	st.version++
}

// Clear removes styles from [q0,q1). With names, only spans with one of
// those names are affected, so a diagnostics writer can clear its own
// marks without touching syntax colours.
func (st *StyleTable) Clear(q0, q1 int, names ...string) {
	if q0 >= q1 {
		return
	}
	var keep map[string]bool
	if len(names) > 0 {
		keep = make(map[string]bool, len(names))
		for _, n := range names {
			keep[n] = true
		}
	}
	if st.cut(q0, q1, keep) {
		st.version++
	}
}

// ClearAll removes every span.
func (st *StyleTable) ClearAll() {
	if len(st.spans) == 0 {
		return
	}
	st.spans = st.spans[:0]
	st.version++
}

// cut removes [q0,q1) from the spans that intersect it (only those whose
// name is in only, when only is not nil), splitting a span that straddles
// the range. It reports whether anything changed.
func (st *StyleTable) cut(q0, q1 int, only map[string]bool) bool {
	changed := false
	out := make([]Span, 0, len(st.spans)+1)
	for _, s := range st.spans {
		if s.Q1 <= q0 || s.Q0 >= q1 || (only != nil && !only[s.Name]) {
			out = append(out, s)
			continue
		}
		changed = true
		if s.Q0 < q0 {
			out = append(out, Span{Q0: s.Q0, Q1: q0, Name: s.Name})
		}
		if s.Q1 > q1 {
			out = append(out, Span{Q0: q1, Q1: s.Q1, Name: s.Name})
		}
	}
	st.spans = out
	return changed
}

// mergeAround merges span i with equal-named neighbours that touch it.
func (st *StyleTable) mergeAround(i int) {
	if i+1 < len(st.spans) && st.spans[i].Q1 == st.spans[i+1].Q0 && st.spans[i].Name == st.spans[i+1].Name {
		st.spans[i].Q1 = st.spans[i+1].Q1
		st.spans = append(st.spans[:i+1], st.spans[i+2:]...)
	}
	if i > 0 && st.spans[i-1].Q1 == st.spans[i].Q0 && st.spans[i-1].Name == st.spans[i].Name {
		st.spans[i-1].Q1 = st.spans[i].Q1
		st.spans = append(st.spans[:i], st.spans[i+1:]...)
	}
}

// inserted shifts the spans for n runes inserted at q0. A span that
// strictly contains q0 grows; a span starting at or after q0 moves. Text
// typed right after a span does not join it.
func (st *StyleTable) inserted(q0, n int) {
	if n <= 0 {
		return
	}
	changed := false
	for i := range st.spans {
		s := &st.spans[i]
		switch {
		case s.Q0 >= q0:
			s.Q0 += n
			s.Q1 += n
			changed = true
		case s.Q1 > q0:
			s.Q1 += n
			changed = true
		}
	}
	if changed {
		st.version++
	}
}

// deleted shifts the spans for the deletion of [q0,q1).
func (st *StyleTable) deleted(q0, q1 int) {
	n := q1 - q0
	if n <= 0 {
		return
	}
	changed := false
	out := st.spans[:0]
	for _, s := range st.spans {
		switch {
		case s.Q1 <= q0:
			// before the deletion
		case s.Q0 >= q1:
			s.Q0 -= n
			s.Q1 -= n
			changed = true
		default:
			changed = true
			if s.Q0 >= q0 {
				s.Q0 = q0
			}
			if s.Q1 >= q1 {
				s.Q1 -= n
			} else {
				s.Q1 = q0
			}
			if s.Q0 >= s.Q1 {
				continue
			}
		}
		out = append(out, s)
	}
	st.spans = out
	if changed {
		st.version++
	}
}
