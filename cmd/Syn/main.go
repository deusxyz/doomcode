// Syn highlights the syntax of Edwood windows.
//
// Usage:
//
//	Syn [-v] [-all] [-delay 100ms]
//	Syn -dump file.go
//
// Syn watches the acme log for windows whose names end in a suffix it
// knows (.go, .md), reads their bodies, and writes styled spans to each
// window's style file; it then follows the window's changes file and
// re-highlights after every pause in editing. Styles are names
// ("keyword", "comment", ...): the editor's theme decides the colours.
//
// Run it from a tag inside Edwood, like acmego: it needs the same
// NAMESPACE. Del on its +Errors window does not stop it; kill the
// process or Kill Syn.
//
// With -all, variables, operators and punctuation are styled too; the
// default themes draw them as plain text, so by default they are left
// out to keep the style writes small.
//
// Syn is part of the justcode project; see docs/05-style-spec.md.
package main

import (
	"flag"
	"log"
	"os"
	"sync"
	"time"

	"9fans.net/go/acme"
)

var (
	verbose = flag.Bool("v", false, "log what is highlighted")
	all     = flag.Bool("all", false, "also style variables, operators and punctuation")
	delay   = flag.Duration("delay", 100*time.Millisecond, "pause after the last edit before re-highlighting")
	dump    = flag.String("dump", "", "highlight this file, print the style write to stdout and exit (for debugging)")
)

func main() {
	flag.Parse()
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	if *dump != "" {
		lang := languageFor(*dump)
		if lang == nil {
			log.Fatalf("Syn: %s: unknown language", *dump)
		}
		text, err := os.ReadFile(*dump)
		if err != nil {
			log.Fatalf("Syn: %v", err)
		}
		os.Stdout.WriteString(styleText(text, highlight(lang, text, *all)))
		return
	}

	var mu sync.Mutex
	workers := map[int]*worker{}
	start := func(id int, name string) {
		lang := languageFor(name)
		if lang == nil {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if _, running := workers[id]; running {
			return
		}
		wk := newWorker(id, lang)
		workers[id] = wk
		go func() {
			wk.run()
			mu.Lock()
			if workers[id] == wk {
				delete(workers, id)
			}
			mu.Unlock()
		}()
	}
	stop := func(id int) {
		mu.Lock()
		defer mu.Unlock()
		if wk, ok := workers[id]; ok {
			close(wk.stop)
			delete(workers, id)
		}
	}

	wins, err := acme.Windows()
	if err != nil {
		log.Fatalf("Syn: %v", err)
	}
	for _, wi := range wins {
		start(wi.ID, wi.Name)
	}

	lr, err := acme.Log()
	if err != nil {
		log.Fatalf("Syn: %v", err)
	}
	for {
		ev, err := lr.Read()
		if err != nil {
			log.Fatalf("Syn: log: %v", err)
		}
		switch ev.Op {
		case "new", "zerox", "get":
			if ev.Name != "" {
				start(ev.ID, ev.Name)
			}
		case "del":
			stop(ev.ID)
		}
	}
}
