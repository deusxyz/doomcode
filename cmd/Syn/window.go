package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"9fans.net/go/acme"
	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
)

// The acme client library only opens the file names it knows, so style and
// changes are opened on our own mount of the acme service.
var (
	fsysOnce sync.Once
	fsys     *client.Fsys
	fsysErr  error
)

func openWinFile(id int, name string, mode uint8) (*client.Fid, error) {
	fsysOnce.Do(func() { fsys, fsysErr = client.MountService("acme") })
	if fsysErr != nil {
		return nil, fsysErr
	}
	return fsys.Open(fmt.Sprintf("%d/%s", id, name), mode)
}

// writeStyle writes st to a style fid in pieces that end at line
// boundaries: 9P splits large writes, and each piece the editor receives
// must parse on its own. The first piece carries the "clear".
func writeStyle(f *client.Fid, st string) error {
	const max = 4096
	for len(st) > 0 {
		n := len(st)
		if n > max {
			n = max
			for n > 0 && st[n-1] != '\n' {
				n--
			}
			if n == 0 { // a single line longer than max: send it whole
				n = len(st)
				for i := 0; i < len(st); i++ {
					if st[i] == '\n' {
						n = i + 1
						break
					}
				}
			}
		}
		if _, err := f.Write([]byte(st[:n])); err != nil {
			return err
		}
		st = st[n:]
	}
	return nil
}

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
func bodyRunes(w *acme.Win) (int, bool) {
	info, err := w.Info()
	if err != nil {
		return 0, false
	}
	return info.BodyLen, true
}

func (wk *worker) run() {
	w, err := acme.Open(wk.id, nil)
	if err != nil {
		log.Printf("Syn: window %d: %v", wk.id, err)
		return
	}
	defer w.CloseFiles()
	styleFid, err := openWinFile(wk.id, "style", plan9.OWRITE)
	if err != nil {
		log.Printf("Syn: window %d: %v (is this Edwood with style support?)", wk.id, err)
		return
	}
	defer styleFid.Close()
	changesFid, err := openWinFile(wk.id, "changes", plan9.OREAD)
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
		if err := writeStyle(styleFid, st); err != nil {
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
				if n, ok := bodyRunes(w); ok && n != doc.Runes() {
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
			if err := writeStyle(styleFid, st); err != nil {
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
