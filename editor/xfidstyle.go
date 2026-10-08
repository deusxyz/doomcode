package main

import (
	"fmt"
	"strconv"
	"strings"

	"9fans.net/go/plan9"
	"github.com/deusxyz/doomcode/editor/file"
)

// The style and changes files of a window directory.
//
// style (read/write) holds the styled spans of the body, "q0 q1 name" per
// line in rune offsets. Writing applies lines atomically: "q0 q1 name"
// sets a span, "clear" removes all, "clear q0 q1 [name...]" removes
// spans (of those names) in a range, "#" starts a comment.
//
// changes (read-only) streams the body's insertions and deletions in the
// event file's format, without intercepting anything and to every reader.
//
// See docs/05-style-spec.md sections 2 and 3 in the justcode project.

// styleString serialises a style table as the style file's contents.
func styleString(st *file.StyleTable) string {
	var sb strings.Builder
	for _, sp := range st.Spans() {
		fmt.Fprintf(&sb, "%d %d %s\n", sp.Q0, sp.Q1, sp.Name)
	}
	return sb.String()
}

// styleOp is one parsed line of a style write.
type styleOp struct {
	clear  bool
	all    bool // clear without a range
	q0, q1 int
	name   string
	names  []string
}

// parseStyleWrite parses the text of a style write. Errors name the line.
func parseStyleWrite(data string) ([]styleOp, error) {
	var ops []styleOp
	for lineno, line := range strings.Split(data, "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "clear" {
			if len(fields) == 1 {
				ops = append(ops, styleOp{clear: true, all: true})
				continue
			}
			if len(fields) < 3 {
				return nil, fmt.Errorf("style: line %d: want \"clear\" or \"clear q0 q1 [name...]\"", lineno+1)
			}
			q0, err0 := strconv.Atoi(fields[1])
			q1, err1 := strconv.Atoi(fields[2])
			if err0 != nil || err1 != nil || q0 < 0 || q1 < q0 {
				return nil, fmt.Errorf("style: line %d: bad range %q %q", lineno+1, fields[1], fields[2])
			}
			ops = append(ops, styleOp{clear: true, q0: q0, q1: q1, names: fields[3:]})
			continue
		}
		if len(fields) != 3 {
			return nil, fmt.Errorf("style: line %d: want \"q0 q1 name\", got %q", lineno+1, line)
		}
		q0, err0 := strconv.Atoi(fields[0])
		q1, err1 := strconv.Atoi(fields[1])
		if err0 != nil || err1 != nil || q0 < 0 || q1 < q0 {
			return nil, fmt.Errorf("style: line %d: bad range %q %q", lineno+1, fields[0], fields[1])
		}
		ops = append(ops, styleOp{q0: q0, q1: q1, name: fields[2]})
	}
	return ops, nil
}

// applyStyleOps applies ops to st, clipping to a text of n runes, and
// returns the range of the text whose styles may have changed.
func applyStyleOps(st *file.StyleTable, ops []styleOp, n int) (lo, hi int) {
	lo, hi = n, 0
	touch := func(q0, q1 int) {
		if q0 < lo {
			lo = q0
		}
		if q1 > hi {
			hi = q1
		}
	}
	for _, op := range ops {
		switch {
		case op.clear && op.all:
			if st.Len() > 0 {
				touch(0, n)
			}
			st.ClearAll()
		case op.clear:
			q1 := op.q1
			if q1 > n {
				q1 = n
			}
			if op.q0 < q1 {
				st.Clear(op.q0, q1, op.names...)
				touch(op.q0, q1)
			}
		default:
			q1 := op.q1
			if q1 > n {
				q1 = n
			}
			if op.q0 < q1 {
				st.Set(op.q0, q1, op.name)
				touch(op.q0, q1)
			}
		}
	}
	return lo, hi
}

// xfidstylewrite handles a write to the style file. The window is locked.
func xfidstylewrite(x *Xfid, w *Window) {
	var fc plan9.Fcall
	ops, err := parseStyleWrite(string(x.fcall.Data))
	if err != nil {
		x.respond(&fc, err)
		return
	}
	w.body.Commit()
	lo, hi := applyStyleOps(w.body.file.Styles(), ops, w.body.file.Nr())
	if lo < hi {
		// Every window showing this file (Zerox clones share it).
		w.body.file.AllObservers(func(i interface{}) {
			if t, ok := i.(*Text); ok && t.what == Body {
				t.Restyle(lo, hi)
			}
		})
		if w.display != nil {
			w.display.Flush()
		}
	}
	fc.Count = x.fcall.Count
	x.respond(&fc, nil)
}

// changeReader is the queue of one open changes file.
type changeReader struct {
	buf    []byte
	x      *Xfid // a read waiting for data, if any
	closed bool  // the window went away
}

// wake answers a pending read: with data if there is any, else by letting
// xfidchangesread report the window shut down (closed) or a flush.
func (r *changeReader) wake(closed bool) {
	if closed {
		r.closed = true
		r.buf = r.buf[:0]
	}
	if x := r.x; x != nil {
		r.x = nil
		x.c <- nil
	}
}

func (w *Window) addChangeReader() *changeReader {
	r := &changeReader{}
	w.changereaders = append(w.changereaders, r)
	return r
}

func (w *Window) delChangeReader(r *changeReader) {
	for i, rr := range w.changereaders {
		if rr == r {
			w.changereaders = append(w.changereaders[:i], w.changereaders[i+1:]...)
			return
		}
	}
}

// Changef appends a change message to every open changes file of w, in
// the event file's format with the owner character first.
func (w *Window) Changef(format string, args ...interface{}) {
	if len(w.changereaders) == 0 {
		return
	}
	owner := byte(w.owner)
	if owner == 0 {
		owner = 'F'
	}
	msg := append([]byte{owner}, fmt.Sprintf(format, args...)...)
	for _, r := range w.changereaders {
		r.buf = append(r.buf, msg...)
		r.wake(false)
	}
}

// xfidchangesread serves a read of a changes file, blocking until there
// is something to report. The window is locked on entry and exit.
func xfidchangesread(x *Xfid, w *Window) {
	var fc plan9.Fcall
	r := x.f.changes
	if r == nil {
		x.respond(&fc, fmt.Errorf("changes file not open"))
		return
	}
	x.flushed = false
	for len(r.buf) == 0 {
		if r.closed {
			x.respond(&fc, fmt.Errorf("window shut down"))
			return
		}
		if x.flushed {
			return
		}
		if r.x != nil {
			x.respond(&fc, fmt.Errorf("changes file already being read"))
			return
		}
		r.x = x
		w.Unlock()
		<-x.c
		w.Lock('F')
	}
	n := len(r.buf)
	if uint32(n) > x.fcall.Count {
		n = int(x.fcall.Count)
	}
	fc.Count = uint32(n)
	fc.Data = r.buf[:n]
	x.respond(&fc, nil)
	r.buf = r.buf[n:]
}
