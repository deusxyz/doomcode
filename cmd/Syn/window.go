package main

import (
	"fmt"
	"log"
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
// highlights the body once, then follows the changes file and
// re-highlights after each burst of edits.
type worker struct {
	id   int
	lang *Language
	stop chan struct{}
}

func newWorker(id int, lang *Language) *worker {
	return &worker{id: id, lang: lang, stop: make(chan struct{})}
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

	last := ""
	refresh := func() {
		body, err := w.ReadAll("body")
		if err != nil {
			return
		}
		st := styleText(body, highlight(wk.lang, body, *all))
		if st == last {
			return
		}
		if err := writeStyle(styleFid, st); err != nil {
			log.Printf("Syn: window %d: style: %v", wk.id, err)
			return
		}
		last = st
		if *verbose {
			log.Printf("Syn: window %d: %d bytes of %s styles", wk.id, len(st), wk.lang.Name)
		}
	}
	refresh()

	// Each message on changes means the body moved; wait for a pause.
	changes := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 8192)
		for {
			if _, err := changesFid.Read(buf); err != nil {
				return // window deleted or Syn stopped
			}
			select {
			case changes <- struct{}{}:
			default:
			}
		}
	}()

	var timer <-chan time.Time
	for {
		select {
		case <-changes:
			timer = time.After(*delay)
		case <-timer:
			timer = nil
			refresh()
		case <-done:
			return
		case <-wk.stop:
			return
		}
	}
}
