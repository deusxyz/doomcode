package main

// Keyboard selection without Shift: an anchor, in the manner of tmux's
// copy mode. Ctrl-B Space drops the anchor at the caret; from then on the
// cursor keys extend the selection between the anchor and the caret until
// something else happens to the selection (typing, Cut/Paste, Esc, a mouse
// click). The caret is the end of the selection that is not the anchor.
//
// Also here: whole-line and word selection, selecting the block inside the
// nearest brackets, indenting and outdenting the selected lines, and the
// "find" and "goto" commands that type into the tag.
//
// See docs/03-keyboard-spec.md sections 3 and 4 in the justcode project.

// caret is the position cursor movement starts from: the selection end
// opposite the anchor, or q0.
func (t *Text) caret() int {
	if t.anchorOn && t.q0 == t.anchor {
		return t.q1
	}
	return t.q0
}

// moveCaret moves the caret to q: with an anchor the selection becomes
// [anchor,q) or [q,anchor); without one the selection collapses to q.
func (t *Text) moveCaret(q int) {
	if q < 0 {
		q = 0
	}
	if n := t.file.Nr(); q > n {
		q = n
	}
	if t.anchorOn {
		if q < t.anchor {
			t.Show(q, t.anchor, true)
		} else {
			t.Show(t.anchor, q, true)
		}
		return
	}
	t.Show(q, q, true)
}

// dropAnchor forgets the anchor; the selection stays as it is.
func (t *Text) dropAnchor() {
	t.anchorOn = false
}

// keyAnchor is the "anchor" action: start extending the selection from
// here. With a selection already present the anchor is its start, so the
// cursor keys move its end.
func (t *Text) keyAnchor() {
	t.TypeCommit()
	t.anchorOn = true
	t.anchor = t.q0
}

// keyCollapse ends anchor mode and leaves the caret alone (Esc).
func (t *Text) keyCollapse() {
	q := t.caret()
	t.dropAnchor()
	t.Show(q, q, true)
}

// keySelectLine selects the whole line (or lines) under the selection; if
// whole lines are already selected it adds the next line.
func (t *Text) keySelectLine() {
	t.TypeCommit()
	t.dropAnchor()
	n := t.file.Nr()
	q0, q1 := t.lineStart(t.q0), t.lineEnd(t.q1)
	if q1 < n {
		q1++ // include the newline
	}
	if q0 == t.q0 && q1 == t.q1 && q1 < n {
		q1 = t.lineEnd(q1)
		if q1 < n {
			q1++
		}
	}
	t.Show(q0, q1, true)
}

// keySelectWord selects the word under the caret as a double click would;
// with a selection present it finds the next occurrence of the selected
// text instead, wrapping around, like Look.
func (t *Text) keySelectWord() {
	t.TypeCommit()
	t.dropAnchor()
	if t.q0 == t.q1 {
		q0, q1 := t.DoubleClick(t.q0, t.q0)
		t.Show(q0, q1, true)
		return
	}
	r := make([]rune, t.q1-t.q0)
	t.file.Read(t.q0, r)
	search(t, r)
}

var blockOpen, blockClose = []rune("([{"), []rune(")]}")

func blockIndex(set []rune, c rune) int {
	for i, r := range set {
		if r == c {
			return i
		}
	}
	return -1
}

// blockAround returns the text inside the nearest pair of brackets that
// encloses [q0,q1). If that is already the selection, the brackets are
// included; if the brackets are already included, the next enclosing pair
// is used.
func (t *Text) blockAround(q0, q1 int) (int, int, bool) {
	// Walk left from q0 to the unmatched opener.
	depth := 0
	open := -1
	for p := q0; p > 0; p-- {
		c := t.file.ReadC(p - 1)
		if blockIndex(blockClose, c) >= 0 {
			depth++
		} else if blockIndex(blockOpen, c) >= 0 {
			if depth == 0 {
				open = p - 1
				break
			}
			depth--
		}
	}
	if open < 0 {
		return 0, 0, false
	}
	want := blockClose[blockIndex(blockOpen, t.file.ReadC(open))]
	// Walk right from q1 to the matching closer.
	depth = 0
	n := t.file.Nr()
	closer := -1
	for p := q1; p < n; p++ {
		c := t.file.ReadC(p)
		if blockIndex(blockOpen, c) >= 0 {
			depth++
		} else if blockIndex(blockClose, c) >= 0 {
			if depth == 0 {
				if c != want {
					return 0, 0, false
				}
				closer = p
				break
			}
			depth--
		}
	}
	if closer < 0 {
		return 0, 0, false
	}
	if q0 == open+1 && q1 == closer {
		// The inside is already selected: include the brackets. The next
		// call scans outward from them and finds the enclosing pair.
		return open, closer + 1, true
	}
	return open + 1, closer, true
}

// keySelectBlock selects the block inside the nearest enclosing brackets,
// growing outwards on repeated use.
func (t *Text) keySelectBlock() {
	t.TypeCommit()
	t.dropAnchor()
	q0, q1 := t.q0, t.q1
	if q0 == q1 && q0 > 0 && blockIndex(blockOpen, t.file.ReadC(q0-1)) >= 0 {
		// Right after an opener: the inside of that bracket.
		if nq0, nq1, ok := t.blockAround(q0, q0); ok {
			t.Show(nq0, nq1, true)
		}
		return
	}
	if q0 == q1 && q0 < t.file.Nr() && blockIndex(blockClose, t.file.ReadC(q0)) >= 0 {
		// Right before a closer: likewise.
		if nq0, nq1, ok := t.blockAround(q0, q0); ok {
			t.Show(nq0, nq1, true)
		}
		return
	}
	if nq0, nq1, ok := t.blockAround(q0, q1); ok {
		t.Show(nq0, nq1, true)
	}
}

// indentString is what one indent level inserts in t.
func (t *Text) indentString() []rune {
	if t.tabexpand {
		r := make([]rune, t.tabstop)
		for i := range r {
			r[i] = ' '
		}
		return r
	}
	return []rune{'\t'}
}

// selectionSpansLines reports whether the selection contains a newline,
// i.e. Tab should indent rather than replace it.
func (t *Text) selectionSpansLines() bool {
	for q := t.q0; q < t.q1; q++ {
		if t.file.ReadC(q) == '\n' {
			return true
		}
	}
	return false
}

// indentLines inserts one indent level at the start of every line touched
// by the selection and leaves those whole lines selected.
func (t *Text) indentLines() {
	t.markUndo()
	t.TypeCommit()
	t.dropAnchor()
	ind := t.indentString()
	q0 := t.lineStart(t.q0)
	q1 := t.q1
	if q1 > q0 && t.file.ReadC(q1-1) == '\n' {
		q1-- // a selection ending at a line start does not indent that line
	}
	for q := q0; q <= q1 && q <= t.file.Nr(); {
		if q < t.file.Nr() || q == q0 {
			t.file.InsertAt(q, ind)
			q1 += len(ind)
		}
		e := t.lineEnd(q)
		if e >= t.file.Nr() {
			break
		}
		q = e + 1
		if q > q1 {
			break
		}
	}
	t.Show(q0, t.lineEnd(q1), true)
	if end := t.lineEnd(q1); end < t.file.Nr() {
		t.SetSelect(q0, end+1)
	}
}

// outdentLines removes one indent level (a tab, or up to tabstop spaces)
// from the start of every line touched by the selection.
func (t *Text) outdentLines() {
	t.markUndo()
	t.TypeCommit()
	t.dropAnchor()
	q0 := t.lineStart(t.q0)
	q1 := t.q1
	if q1 > q0 && t.file.ReadC(q1-1) == '\n' {
		q1--
	}
	if q1 < q0 {
		q1 = q0
	}
	for q := q0; ; {
		del := 0
		if q < t.file.Nr() && t.file.ReadC(q) == '\t' {
			del = 1
		} else {
			for del < t.tabstop && q+del < t.file.Nr() && t.file.ReadC(q+del) == ' ' {
				del++
			}
		}
		if del > 0 {
			t.file.DeleteAt(q, q+del)
			q1 -= del
			if q1 < q {
				q1 = q
			}
		}
		e := t.lineEnd(q)
		if e >= t.file.Nr() || e+1 > q1 {
			break
		}
		q = e + 1
	}
	end := t.lineEnd(q1)
	if end < t.file.Nr() {
		end++
	}
	t.Show(q0, end, true)
}

// typeIntoTag puts s at the end of the window's tag and the caret after
// it, marks s as "typed" so that Esc selects it together with what the
// user types next, and moves the focus to the tag. find and goto use it:
// the user types the rest, presses Esc, then ^E or ^O.
func (t *Text) typeIntoTag(s string) {
	if t.w == nil {
		return
	}
	tag := &t.w.tag
	tag.Commit()
	n := tag.file.Nr()
	r := []rune(s)
	if n > 0 && tag.file.ReadC(n-1) != ' ' {
		r = append([]rune{' '}, r...)
	}
	tag.file.InsertAt(n, r)
	tag.eq0 = n + len(r) - len([]rune(s))
	end := n + len(r)
	tag.SetSelect(end, end)
	t.w.Commit(tag)
	global.setFocus(tag)
}

// keyFind: with a selection, find its next occurrence (like Look); without
// one, type "Look " into the tag for the user to complete.
func (t *Text) keyFind() {
	t.TypeCommit()
	t.dropAnchor()
	if t.q0 != t.q1 {
		r := make([]rune, t.q1-t.q0)
		t.file.Read(t.q0, r)
		search(t, r)
		return
	}
	t.typeIntoTag("Look ")
}

// keyGoto types ":" into the tag; the user completes the address (27,
// /regexp/, $) and presses Esc then ^O.
func (t *Text) keyGoto() {
	t.TypeCommit()
	t.dropAnchor()
	t.typeIntoTag(":")
}
