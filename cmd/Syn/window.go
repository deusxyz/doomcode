package main

import (
	"log"
	"strings"
	"time"

	"9fans.net/go/acme"
	"github.com/deusxyz/doomcode/internal/acmefs"
)

// A worker keeps one window's style file in step with its body: it
// highlights the body once, then follows the changes file, applies each
// change to its copy of the text and the parse tree, and after a pause
// rewrites only the styles of the region that changed.
type worker struct {
	id   int
	lang *Language
	stop chan struct{}
}

func newWorker(id int, lang *Language) *worker {
	return &worker{id: id, lang: lang, stop: make(chan struct{})}
}

// bodyRunes reads the body length in runes from the window's ctl line.
func bodyRunes(id int) (int, bool) {
	n, err := acmefs.BodyRunes(id)
	return n, err == nil
}

func (wk *worker) run() {
	w, err := acme.Open(wk.id, nil)
	if err != nil {
		log.Printf("Syn: window %d: %v", wk.id, err)
		return
	}
	defer w.CloseFiles()
	styleFid, err := acmefs.OpenStyle(wk.id)
	if err != nil {
		log.Printf("Syn: window %d: %v (is this Edwood with style support?)", wk.id, err)
		return
	}
	defer styleFid.Close()
	changesFid, err := acmefs.OpenChanges(wk.id)
	if err != nil {
		log.Printf("Syn: window %d: %v", wk.id, err)
		return
	}
	defer changesFid.Close()

	var doc *document
	defer func() {
		if doc != nil {
			doc.close()
		}
	}()

	// resync reads the whole body, parses it afresh and rewrites all
	// styles. It is the starting point and the fallback whenever the
	// incremental state is in doubt.
	resync := func(why string) bool {
		body, err := w.ReadAll("body")
		if err != nil {
			return false
		}
		if doc != nil {
			doc.close()
			doc = nil
		}
		d, err := newDocument(wk.lang, body)
		if err != nil {
			log.Printf("Syn: window %d: %v", wk.id, err)
			return false
		}
		doc = d
		st := doc.flush(true, *all)
		if err := acmefs.WriteStyle(styleFid, st); err != nil {
			log.Printf("Syn: window %d: style: %v", wk.id, err)
			return false
		}
		if *verbose {
			log.Printf("Syn: window %d: full %s highlight (%s), %d bytes", wk.id, wk.lang.Name, why, len(st))
		}
		return true
	}
	if !resync("open") {
		return
	}

	// fetch gets the text of a long insertion the editor did not include
	// in its message, from the body as it is now. Only valid while no
	// later change has moved it, so callers resync on any doubt.
	fetch := func(c *change) bool {
		if err := w.Addr("#%d,#%d", c.Q0, c.Q1); err != nil {
			return false
		}
		b, err := w.ReadAll("xdata")
		if err != nil {
			return false
		}
		c.Text, c.HasText = b, true
		return true
	}

	// Change messages arrive on a goroutine; the worker applies them and
	// flushes after a pause.
	msgs := make(chan change, 64)
	bad := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 8192)
		var pending []byte
		for {
			n, err := changesFid.Read(buf)
			if err != nil {
				return // window deleted or Syn stopped
			}
			pending = append(pending, buf[:n]...)
			cs, rest, err := parseChanges(pending)
			if err != nil {
				bad <- err
				return
			}
			pending = rest
			for _, c := range cs {
				msgs <- c
			}
		}
	}()

	var timer <-chan time.Time
	needResync := ""
	for {
		select {
		case c := <-msgs:
			if needResync != "" {
				break // the next flush will resync anyway
			}
			if c.Insert && !c.HasText {
				// Long insertion: the text is only right if nothing has
				// moved since; with more messages queued, resync instead.
				if len(msgs) > 0 || !fetch(&c) {
					needResync = "long insertion"
					break
				}
			}
			if !doc.apply(c) {
				needResync = "change did not fit"
			}
			timer = time.After(*delay)
		case <-timer:
			timer = nil
			if needResync == "" && doc.edits >= 200 {
				// Cheap consistency check now and then: the body length.
				if n, ok := bodyRunes(wk.id); ok && n != doc.Runes() {
					needResync = "length mismatch"
				}
				doc.edits = 0
			}
			if needResync != "" {
				why := needResync
				needResync = ""
				if !resync(why) {
					return
				}
				break
			}
			st := doc.flush(false, *all)
			if st == "" {
				break
			}
			if err := acmefs.WriteStyle(styleFid, st); err != nil {
				log.Printf("Syn: window %d: style: %v", wk.id, err)
				return
			}
			if *verbose {
				first := st
				if i := strings.IndexByte(st, '\n'); i >= 0 {
					first = st[:i]
				}
				log.Printf("Syn: window %d: %s, %d lines", wk.id, first, strings.Count(st, "\n")-1)
			}
		case err := <-bad:
			log.Printf("Syn: window %d: %v; resyncing", wk.id, err)
			if !resync("bad message") {
				return
			}
			return // the reader goroutine has quit; let the log restart us on the next event
		case <-done:
			return
		case <-wk.stop:
			return
		}
	}
}
