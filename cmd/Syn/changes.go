package main

import (
	"bytes"
	"fmt"
	"strconv"
	"unicode/utf8"
)

// A change is one message from a window's changes file: text inserted
// at [Q0,Q1) or deleted from [Q0,Q1), in rune offsets of the body as it
// was just before the change. Text is the inserted text, or nil when the
// editor left it out (more than 256 runes); the reader then has to fetch
// it from the body itself.
type change struct {
	Insert bool
	Q0, Q1 int
	Text   []byte
	// HasText is false when the message carried no text although Insert
	// is true (the editor omits long insertions).
	HasText bool
}

// parseChanges extracts complete messages from buf and returns them with
// the unconsumed remainder. The format is the event file's:
//
//	<origin><type> q0 q1 flag nrunes text\n
//
// where text holds nrunes runes (it may contain newlines) and is empty
// with nrunes == 0 for deletions and for long insertions.
func parseChanges(buf []byte) ([]change, []byte, error) {
	var out []change
	for {
		if len(buf) < 2 {
			return out, buf, nil
		}
		typ := buf[1]
		// Four decimal fields each followed by a space.
		p := 2
		var nums [4]int
		for i := 0; i < 4; i++ {
			sp := bytes.IndexByte(buf[p:], ' ')
			if sp < 0 {
				return out, buf, nil // incomplete header
			}
			n, err := strconv.Atoi(string(buf[p : p+sp]))
			if err != nil {
				return out, nil, fmt.Errorf("changes: bad header %q", buf[:p+sp])
			}
			nums[i] = n
			p += sp + 1
		}
		nr := nums[3]
		// nr runes of text, then a newline.
		end := p
		for i := 0; i < nr; i++ {
			if end >= len(buf) {
				return out, buf, nil // text not all here yet
			}
			_, size := utf8.DecodeRune(buf[end:])
			if size == 0 || (buf[end] >= 0x80 && !utf8.FullRune(buf[end:])) {
				return out, buf, nil
			}
			end += size
		}
		if end >= len(buf) {
			return out, buf, nil // newline not here yet
		}
		if buf[end] != '\n' {
			return out, nil, fmt.Errorf("changes: message not terminated: %q", buf[:end+1])
		}
		c := change{Q0: nums[0], Q1: nums[1]}
		switch typ {
		case 'I':
			c.Insert = true
			c.HasText = nr > 0 || nums[1] == nums[0]
			c.Text = append([]byte(nil), buf[p:end]...)
		case 'D':
		default:
			return out, nil, fmt.Errorf("changes: unknown message type %q", typ)
		}
		out = append(out, c)
		buf = buf[end+1:]
	}
}
