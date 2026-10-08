package main

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// fmtTimeout bounds a formatter run; Put waits for it.
const fmtTimeout = 10 * time.Second

// runFormatter pipes src through argv in dir and returns its output. A
// non-zero exit or a timeout is an error; stderr comes back for the
// warning either way.
func runFormatter(argv []string, dir string, src string) (out string, stderr string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), fmtTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(src)
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se
	err = cmd.Run()
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	return so.String(), se.String(), err
}

// formatBody runs argv over the body and replaces the body with the
// output when it differs. It reports whether the body changed. Errors go
// to +Errors; the body is left alone then.
func (w *Window) formatBody(argv []string) bool {
	if len(argv) == 0 || w.body.file.IsDir() {
		return false
	}
	w.Commit(&w.body)
	name := w.body.file.Name()
	src := w.body.file.String()
	out, stderr, err := runFormatter(argv, w.body.DirName(""), src)
	if err != nil {
		warning(nil, "Fmt: %s: %s: %v\n%s", name, strings.Join(argv, " "), err, stderr)
		return false
	}
	if stderr != "" {
		warning(nil, "Fmt: %s: %s", name, stderr)
	}
	if out == src {
		return false
	}
	w.body.replaceAll(out)
	return true
}

// replaceAll makes the text s, touching only the span between the common
// prefix and suffix of the old and new text so that the selection, the
// scroll position and the styles outside it survive. The selection is
// mapped across the change: positions before it stay, positions after it
// shift, positions strictly inside it go to the end of the new span (so a
// selection covering the changed text ends up covering the new text).
func (t *Text) replaceAll(s string) {
	old := []rune(t.file.String())
	nw := []rune(s)
	p := 0
	for p < len(old) && p < len(nw) && old[p] == nw[p] {
		p++
	}
	sfx := 0
	for sfx < len(old)-p && sfx < len(nw)-p && old[len(old)-1-sfx] == nw[len(nw)-1-sfx] {
		sfx++
	}
	oldEnd, newEnd := len(old)-sfx, len(nw)-sfx
	remap := func(q int) int {
		switch {
		case q <= p:
			return q
		case q >= oldEnd:
			return q + (len(nw) - len(old))
		}
		return newEnd
	}
	q0, q1 := remap(t.q0), remap(t.q1)
	t.markUndo()
	t.Delete(p, oldEnd, true)
	t.Insert(p, nw[p:newEnd], true)
	t.Show(q0, q1, true)
}
