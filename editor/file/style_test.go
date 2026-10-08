package file

import (
	"fmt"
	"strings"
	"testing"
)

func spansString(st *StyleTable) string {
	var parts []string
	for _, s := range st.spans {
		parts = append(parts, fmt.Sprintf("%d-%d:%s", s.Q0, s.Q1, s.Name))
	}
	return strings.Join(parts, " ")
}

func wantSpans(t *testing.T, what string, st *StyleTable, want string) {
	t.Helper()
	if got := spansString(st); got != want {
		t.Errorf("%s: spans %q; want %q", what, got, want)
	}
}

func TestStyleTableSetAndClear(t *testing.T) {
	var st StyleTable
	st.Set(10, 20, "keyword")
	st.Set(30, 40, "string")
	wantSpans(t, "two spans", &st, "10-20:keyword 30-40:string")

	st.Set(0, 5, "comment") // before everything
	wantSpans(t, "prepend", &st, "0-5:comment 10-20:keyword 30-40:string")

	st.Set(15, 35, "type") // overlaps two: trims both
	wantSpans(t, "overlap", &st, "0-5:comment 10-15:keyword 15-35:type 35-40:string")

	st.Set(12, 13, "number") // inside keyword: splits it
	wantSpans(t, "split", &st, "0-5:comment 10-12:keyword 12-13:number 13-15:keyword 15-35:type 35-40:string")

	st.Set(5, 10, "comment") // touches the comment: merged
	wantSpans(t, "merge right", &st, "0-10:comment 10-12:keyword 12-13:number 13-15:keyword 15-35:type 35-40:string")

	st.Set(20, 20, "x") // empty range ignored
	st.Set(1, 2, "")    // empty name ignored
	wantSpans(t, "ignored", &st, "0-10:comment 10-12:keyword 12-13:number 13-15:keyword 15-35:type 35-40:string")

	st.Clear(11, 14) // trims keyword, removes number, trims keyword
	wantSpans(t, "clear middle", &st, "0-10:comment 10-11:keyword 14-15:keyword 15-35:type 35-40:string")

	st.Clear(0, 100, "type", "string") // only those names
	wantSpans(t, "clear by name", &st, "0-10:comment 10-11:keyword 14-15:keyword")

	v := st.Version()
	st.Clear(50, 60) // nothing there: no version bump
	if st.Version() != v {
		t.Errorf("Clear of an empty range bumped the version")
	}
	st.ClearAll()
	wantSpans(t, "clear all", &st, "")
	if st.Len() != 0 {
		t.Errorf("Len after ClearAll = %d", st.Len())
	}
}

func TestStyleTableAtAndRange(t *testing.T) {
	var st StyleTable
	st.Set(10, 20, "keyword")
	st.Set(20, 30, "string")
	for _, tc := range []struct {
		q    int
		name string
		ok   bool
	}{
		{9, "", false}, {10, "keyword", true}, {19, "keyword", true}, {20, "string", true}, {29, "string", true}, {30, "", false},
	} {
		s, ok := st.At(tc.q)
		if ok != tc.ok || (ok && s.Name != tc.name) {
			t.Errorf("At(%d) = %v,%v; want %q,%v", tc.q, s, ok, tc.name, tc.ok)
		}
	}
	got := st.Range(15, 25)
	if len(got) != 2 || got[0] != (Span{15, 20, "keyword"}) || got[1] != (Span{20, 25, "string"}) {
		t.Errorf("Range(15,25) = %v", got)
	}
	if got := st.Range(0, 10); len(got) != 0 {
		t.Errorf("Range(0,10) = %v; want none", got)
	}
}

func TestStyleTableInserted(t *testing.T) {
	var st StyleTable
	st.Set(10, 20, "a")
	st.Set(30, 40, "b")

	st.inserted(0, 5) // before: everything shifts
	wantSpans(t, "insert before", &st, "15-25:a 35-45:b")

	st.inserted(20, 3) // strictly inside a: a grows
	wantSpans(t, "insert inside", &st, "15-28:a 38-48:b")

	st.inserted(28, 2) // at the end of a: a does not grow, b shifts
	wantSpans(t, "insert at end", &st, "15-28:a 40-50:b")

	st.inserted(15, 1) // at the start of a: a moves
	wantSpans(t, "insert at start", &st, "16-29:a 41-51:b")

	st.inserted(100, 7) // after everything
	wantSpans(t, "insert after", &st, "16-29:a 41-51:b")
}

func TestStyleTableDeleted(t *testing.T) {
	var st StyleTable
	st.Set(10, 20, "a")
	st.Set(30, 40, "b")
	st.Set(50, 60, "c")

	st.deleted(0, 5) // before: shift
	wantSpans(t, "delete before", &st, "5-15:a 25-35:b 45-55:c")

	st.deleted(7, 9) // inside a: shrinks
	wantSpans(t, "delete inside", &st, "5-13:a 23-33:b 43-53:c")

	st.deleted(10, 25) // tail of a and head of b
	wantSpans(t, "delete across", &st, "5-10:a 10-18:b 28-38:c")

	st.deleted(9, 20) // removes the rest of b entirely, trims a
	wantSpans(t, "delete covering a span", &st, "5-9:a 17-27:c")

	st.deleted(0, 100)
	wantSpans(t, "delete everything", &st, "")
}

func TestObservableBufferKeepsStyles(t *testing.T) {
	e := MakeObservableEditableBuffer("", []rune("hello world"))
	st := e.Styles()
	st.Set(0, 5, "a")
	st.Set(6, 11, "b")

	e.InsertAt(0, []rune("XX")) // "XXhello world"
	wantSpans(t, "after insert", st, "2-7:a 8-13:b")

	e.DeleteAt(0, 3) // "ello world"
	wantSpans(t, "after delete", st, "0-4:a 5-10:b")

	e.InsertAt(2, []rune("-")) // inside a: "el-lo world"
	wantSpans(t, "after insert inside", st, "0-5:a 6-11:b")

	e.ResetBuffer()
	wantSpans(t, "after reset", st, "")
}
