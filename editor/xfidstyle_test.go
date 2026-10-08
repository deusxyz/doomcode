package main

import (
	"strings"
	"testing"

	"9fans.net/go/plan9"
	"github.com/deusxyz/doomcode/editor/file"
)

func TestParseStyleWrite(t *testing.T) {
	ops, err := parseStyleWrite("# comment\n0 5 keyword\n\nclear\nclear 10 20\nclear 30 40 error warning # trailing\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 4 {
		t.Fatalf("got %d ops: %+v", len(ops), ops)
	}
	if ops[0].clear || ops[0].q0 != 0 || ops[0].q1 != 5 || ops[0].name != "keyword" {
		t.Errorf("op0 = %+v", ops[0])
	}
	if !ops[1].clear || !ops[1].all {
		t.Errorf("op1 = %+v", ops[1])
	}
	if !ops[2].clear || ops[2].all || ops[2].q0 != 10 || ops[2].q1 != 20 || len(ops[2].names) != 0 {
		t.Errorf("op2 = %+v", ops[2])
	}
	if !ops[3].clear || ops[3].q0 != 30 || ops[3].q1 != 40 || strings.Join(ops[3].names, ",") != "error,warning" {
		t.Errorf("op3 = %+v", ops[3])
	}
	for _, bad := range []string{"1 2", "x 2 k", "5 2 k", "-1 2 k", "clear 1", "clear a b", "1 2 3 4"} {
		if _, err := parseStyleWrite(bad); err == nil {
			t.Errorf("parseStyleWrite(%q) succeeded", bad)
		}
	}
}

func TestApplyStyleOps(t *testing.T) {
	var st file.StyleTable
	ops, _ := parseStyleWrite("0 5 a\n5 100 b\n") // second clipped to the text
	lo, hi := applyStyleOps(&st, ops, 20)
	if lo != 0 || hi != 20 {
		t.Errorf("touched %d,%d; want 0,20", lo, hi)
	}
	if got := styleString(&st); got != "0 5 a\n5 20 b\n" {
		t.Errorf("table %q", got)
	}
	ops, _ = parseStyleWrite("clear 2 3\n")
	lo, hi = applyStyleOps(&st, ops, 20)
	if lo != 2 || hi != 3 || styleString(&st) != "0 2 a\n3 5 a\n5 20 b\n" {
		t.Errorf("after clear: %d,%d %q", lo, hi, styleString(&st))
	}
	ops, _ = parseStyleWrite("clear 0 20 b\n")
	applyStyleOps(&st, ops, 20)
	if styleString(&st) != "0 2 a\n3 5 a\n" {
		t.Errorf("after clear by name: %q", styleString(&st))
	}
	ops, _ = parseStyleWrite("clear\n")
	lo, hi = applyStyleOps(&st, ops, 20)
	if lo != 0 || hi != 20 || st.Len() != 0 {
		t.Errorf("after clear all: %d,%d len %d", lo, hi, st.Len())
	}
	// Clearing an empty table touches nothing.
	lo, hi = applyStyleOps(&st, ops, 20)
	if lo < hi {
		t.Errorf("clear of empty table touched %d,%d", lo, hi)
	}
}

func makeStyleXfid(w *Window, q uint64, data string) (*Xfid, *mockResponder) {
	mr := new(mockResponder)
	x := &Xfid{
		fcall: plan9.Fcall{Data: []byte(data), Count: 8192},
		f:     &Fid{qid: plan9.Qid{Path: QID(0, q)}, w: w},
		fs:    mr,
	}
	return x, mr
}

func TestXfidStyleWriteAndRead(t *testing.T) {
	w := NewWindow().initHeadless(nil)
	w.body.file = file.MakeObservableEditableBuffer("", []rune("func main() {}\n"))
	w.col = new(Column)
	fr := &recordingFrame{nchars: 15}
	w.body.fr = fr
	w.body.what = Body
	w.body.w = w
	w.body.file.AddObserver(&w.body) // as Window.Init does
	global.styles = nil
	defer func() { global.styles = nil }()

	x, mr := makeStyleXfid(w, QWstyle, "0 4 keyword\n5 9 function\n")
	xfidwrite(x)
	if mr.err != nil {
		t.Fatalf("write: %v", mr.err)
	}
	if mr.fcall.Count != uint32(len(x.fcall.Data)) {
		t.Errorf("Count %d; want %d", mr.fcall.Count, len(x.fcall.Data))
	}
	if len(fr.restyles) != 1 || fr.restyles[0].p0 != 0 || fr.restyles[0].p1 != 9 {
		t.Errorf("restyle calls: %+v; want one for 0..9", fr.restyles)
	}

	x, mr = makeStyleXfid(w, QWstyle, "")
	xfidread(x)
	if mr.err != nil {
		t.Fatalf("read: %v", mr.err)
	}
	if got := string(mr.fcall.Data); got != "0 4 keyword\n5 9 function\n" {
		t.Errorf("read %q", got)
	}

	// A bad line applies nothing and reports the line.
	x, mr = makeStyleXfid(w, QWstyle, "clear\n10 11 x\nbogus\n")
	xfidwrite(x)
	if mr.err == nil || !strings.Contains(mr.err.Error(), "line 3") {
		t.Errorf("bad write error = %v", mr.err)
	}
	if w.body.file.Styles().Len() != 2 {
		t.Errorf("bad write changed the table: %q", styleString(w.body.file.Styles()))
	}

	// Styles follow edits made afterwards.
	w.body.file.InsertAt(0, []rune("// c\n"))
	x, mr = makeStyleXfid(w, QWstyle, "")
	xfidread(x)
	if got := string(mr.fcall.Data); got != "5 9 keyword\n10 14 function\n" {
		t.Errorf("after insert: %q", got)
	}
}

func TestXfidChanges(t *testing.T) {
	w := NewWindow().initHeadless(nil)
	w.body.file = file.MakeObservableEditableBuffer("", []rune("hello"))
	w.col = new(Column)
	w.body.fr = &recordingFrame{nchars: 5}
	w.body.what = Body
	w.body.w = w
	w.body.file.AddObserver(&w.body) // as Window.Init does
	w.body.nofill = true             // the mock frame never fills
	w.owner = 'K'

	// Nothing is queued without a reader.
	w.body.file.InsertAt(5, []rune("!"))
	if len(w.changereaders) != 0 {
		t.Fatal("reader list not empty")
	}

	// Two readers each get every message.
	r1 := w.addChangeReader()
	r2 := w.addChangeReader()
	w.body.file.InsertAt(0, []rune("ab"))
	w.body.file.DeleteAt(0, 1)
	want := "KI0 2 0 2 ab\nKD0 1 0 0 \n"
	if string(r1.buf) != want || string(r2.buf) != want {
		t.Errorf("queues %q / %q; want %q", r1.buf, r2.buf, want)
	}

	// A read drains a queue; a short read leaves the rest.
	x, mr := makeStyleXfid(w, QWchanges, "")
	x.f.changes = r1
	x.fcall.Count = 7
	xfidchangesread(x, w)
	if mr.err != nil || string(mr.fcall.Data) != "KI0 2 0" {
		t.Errorf("short read: %v %q", mr.err, mr.fcall.Data)
	}
	x, mr = makeStyleXfid(w, QWchanges, "")
	x.f.changes = r1
	xfidchangesread(x, w)
	if string(mr.fcall.Data) != " 2 ab\nKD0 1 0 0 \n" || len(r1.buf) != 0 {
		t.Errorf("rest: %q, left %q", mr.fcall.Data, r1.buf)
	}
	// Tag edits are not reported.
	w.tag.file = file.MakeObservableEditableBuffer("", nil)
	w.tag.w = w
	w.tag.what = Tag
	w.tag.file.AddObserver(&w.tag)
	w.tag.file.InsertAt(0, []rune("x"))
	if len(r1.buf) != 0 {
		t.Errorf("tag edit reported: %q", r1.buf)
	}

	// Removing a reader stops its deliveries; the other keeps them.
	w.delChangeReader(r2)
	w.body.file.InsertAt(0, []rune("z"))
	if len(w.changereaders) != 1 || string(r1.buf) != "KI0 1 0 1 z\n" {
		t.Errorf("after del: readers %d, r1 %q", len(w.changereaders), r1.buf)
	}

	// A deleted window answers a read with an error: Window.Delete marks
	// every reader closed and wakes pending ones.
	r1.buf = nil
	r1.wake(true)
	x, mr = makeStyleXfid(w, QWchanges, "")
	x.f.changes = r1
	xfidchangesread(x, w)
	if mr.err == nil || !strings.Contains(mr.err.Error(), "shut down") {
		t.Errorf("read after Delete: %v", mr.err)
	}
}
