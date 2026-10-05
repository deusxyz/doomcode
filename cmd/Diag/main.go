// Diag puts acme-lsp's diagnostics into the windows they belong to.
//
// Usage:
//
//	Diag [-v] [-style error] [-window /LSP/Diagnostics]
//
// acme-lsp collects diagnostics from language servers into a window
// named /LSP/Diagnostics, one per line:
//
//	/abs/path/file.go:12.5,12.9: message
//
// Diag follows that window (through its changes file) and, for every file
// mentioned that is open in Edwood, writes the ranges to the window's
// style file as error (or warning/hint when the message says so),
// clearing only those styles, so syntax colours written by Syn stay. Files
// whose diagnostics disappear are cleared too.
//
// The Diagnostics window does not carry the LSP severity, so -style sets
// what a plain message is drawn as. Run Diag from a tag inside Edwood, with
// acme-lsp already running; it needs the same NAMESPACE.
//
// Diag is part of the justcode project; see docs/05-style-spec.md.
package main

import (
	"flag"
	"log"
	"os"
	"time"

	"9fans.net/go/acme"
	"justcode/internal/acmefs"
)

var (
	verbose    = flag.Bool("v", false, "log what is marked")
	defStyle   = flag.String("style", "error", "style for diagnostics whose severity cannot be told from the message")
	diagWindow = flag.String("window", "/LSP/Diagnostics", "name of acme-lsp's diagnostics window")
	delay      = flag.Duration("delay", 300*time.Millisecond, "pause after the window changes before re-reading it")
)

func main() {
	flag.Parse()
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	for {
		id, ok := findWindow(*diagWindow)
		if !ok {
			time.Sleep(2 * time.Second)
			continue
		}
		if *verbose {
			log.Printf("Diag: following window %d (%s)", id, *diagWindow)
		}
		follow(id)
		// The window went away: forget what we marked and wait for a new one.
		clearAll()
		time.Sleep(time.Second)
	}
}

// findWindow returns the id of the window named name.
func findWindow(name string) (int, bool) {
	wins, err := acme.Windows()
	if err != nil {
		log.Fatalf("Diag: %v", err)
	}
	for _, w := range wins {
		if w.Name == name {
			return w.ID, true
		}
	}
	return 0, false
}

// marked remembers which files currently carry diagnostics, so that a
// file dropping out of the window gets cleared.
var marked = map[string]bool{}

// follow reads the diagnostics window until it is deleted, re-applying
// its contents after every pause in its changes.
func follow(id int) {
	w, err := acme.Open(id, nil)
	if err != nil {
		log.Printf("Diag: %v", err)
		return
	}
	defer w.CloseFiles()
	changes, err := acmefs.OpenChanges(id)
	if err != nil {
		log.Printf("Diag: %v (is this Edwood with style support?)", err)
		return
	}
	defer changes.Close()

	apply := func() {
		body, err := w.ReadAll("body")
		if err != nil {
			return
		}
		applyDiagnostics(string(body))
	}
	apply()

	changed := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 8192)
		for {
			if _, err := changes.Read(buf); err != nil {
				return
			}
			select {
			case changed <- struct{}{}:
			default:
			}
		}
	}()
	var timer <-chan time.Time
	for {
		select {
		case <-changed:
			timer = time.After(*delay)
		case <-timer:
			timer = nil
			apply()
		case <-done:
			return
		}
	}
}

// applyDiagnostics writes the diagnostics in body to the open windows.
func applyDiagnostics(body string) {
	byFile := parseDiagnostics(body, *defStyle)
	wins, err := acme.Windows()
	if err != nil {
		return
	}
	byName := map[string]int{}
	for _, wi := range wins {
		byName[wi.Name] = wi.ID
	}
	// Files that had marks and have none now are cleared.
	for file := range marked {
		if _, still := byFile[file]; !still {
			if id, open := byName[file]; open {
				writeWindow(id, nil)
			}
			delete(marked, file)
		}
	}
	for file, diags := range byFile {
		id, open := byName[file]
		if !open {
			continue
		}
		writeWindow(id, diags)
		marked[file] = true
	}
}

// writeWindow writes the style spans for diags into window id; nil diags
// just clears Diag's styles there.
func writeWindow(id int, diags []diagnostic) {
	w, err := acme.Open(id, nil)
	if err != nil {
		return
	}
	defer w.CloseFiles()
	body, err := w.ReadAll("body")
	if err != nil {
		return
	}
	f, err := acmefs.OpenStyle(id)
	if err != nil {
		log.Printf("Diag: window %d: %v", id, err)
		return
	}
	defer f.Close()
	st := styleWrite(body, diags)
	if err := acmefs.WriteStyle(f, st); err != nil {
		log.Printf("Diag: window %d: %v", id, err)
		return
	}
	if *verbose {
		log.Printf("Diag: window %d: %d diagnostic(s)", id, len(diags))
	}
}

// clearAll removes Diag's marks from every window that has them.
func clearAll() {
	wins, err := acme.Windows()
	if err != nil {
		return
	}
	for _, wi := range wins {
		if marked[wi.Name] {
			writeWindow(wi.ID, nil)
		}
	}
	marked = map[string]bool{}
}
