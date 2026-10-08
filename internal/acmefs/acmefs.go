// Package acmefs opens window files that the 9fans.net/go/acme library does
// not know about (Edwood's style and changes) and writes style files in
// pieces the editor can parse.
package acmefs

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"9fans.net/go/plan9"
	"9fans.net/go/plan9/client"
)

var (
	fsysOnce sync.Once
	fsys     *client.Fsys
	fsysErr  error
)

// OpenWinFile opens name in window id's directory with the given 9P mode
// (plan9.OREAD, OWRITE, ORDWR) on a private mount of the acme service.
func OpenWinFile(id int, name string, mode uint8) (*client.Fid, error) {
	fsysOnce.Do(func() { fsys, fsysErr = client.MountService("acme") })
	if fsysErr != nil {
		return nil, fsysErr
	}
	return fsys.Open(fmt.Sprintf("%d/%s", id, name), mode)
}

// OpenStyle opens window id's style file for writing.
func OpenStyle(id int) (*client.Fid, error) { return OpenWinFile(id, "style", plan9.OWRITE) }

// OpenChanges opens window id's changes file for reading.
func OpenChanges(id int) (*client.Fid, error) { return OpenWinFile(id, "changes", plan9.OREAD) }

// WriteStyle writes st to a style fid in pieces that end at line
// boundaries: 9P splits large writes, and each piece the editor receives
// is applied on its own and must parse on its own.
func WriteStyle(f *client.Fid, st string) error {
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

// BodyRunes returns the length in runes of window id's body, the third
// field of its ctl line.
func BodyRunes(id int) (int, error) {
	f, err := OpenWinFile(id, "ctl", plan9.OREAD)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	buf := make([]byte, 256)
	n, err := f.ReadAt(buf, 0)
	if err != nil && err != io.EOF {
		return 0, err
	}
	fields := strings.Fields(string(buf[:n]))
	if len(fields) < 3 {
		return 0, fmt.Errorf("window %d: short ctl line %q", id, buf[:n])
	}
	return strconv.Atoi(fields[2])
}
