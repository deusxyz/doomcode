package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The keys file lets the user change key bindings without rebuilding.
//
// Location: $EDWOOD_KEYS if set, otherwise $XDG_CONFIG_HOME/edwood/keys,
// otherwise $HOME/.config/edwood/keys. A missing file means the built-in
// bindings (defaultBindings in keys.go) are used unchanged.
//
// Format: one binding per line, "key action", where key is spelled as
// ParseKey accepts ("C-s", "Cmd-s", "Left", "F3", "0xF800") and action is a
// name from actionTable, or "run" followed by a command, which the key
// then runs like button 2, or "tag" followed by text, which it types into
// the tag (keyrun.go). "prefix key action" binds the key after the
// Ctrl-B prefix instead. "key -" removes a binding so the key is typed as
// text again. Blank lines are ignored and # starts a comment that runs to
// the end of the line. The file
// is applied on top of the defaults, so it only needs to list changes.
//
// The Keys command (exec.go) lists the bindings, reloads the file and
// documents the actions.

// keysFilePath returns the path of the keys file, see above.
func keysFilePath() string {
	if p := os.Getenv("EDWOOD_KEYS"); p != "" {
		return p
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "edwood", "keys")
	}
	home := os.Getenv("HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return filepath.Join(home, ".config", "edwood", "keys")
}

// loadKeysText applies the bindings read from r to km (direct keys) and
// pkm (keys after the Ctrl-B prefix). Lines that cannot be applied are
// reported in errs, one per line, and skipped; the valid lines are still
// applied. n is the number of lines applied.
func loadKeysText(km, pkm Keymap, r io.Reader) (n int, errs []error) {
	sc := bufio.NewScanner(r)
	for lineno := 1; sc.Scan(); lineno++ {
		line := sc.Text()
		if i := strings.Index(line, "#"); i >= 0 { // comment to end of line
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		target := km
		if len(fields) >= 3 && fields[0] == "prefix" {
			target, fields = pkm, fields[1:]
		}
		// "key run command words..." and "key tag text..." bind the key
		// to a command (keyrun.go).
		if len(fields) >= 3 && (fields[1] == "run" || fields[1] == "tag") {
			fields = []string{fields[0], strings.Join(fields[1:], " ")}
		}
		if len(fields) != 2 {
			errs = append(errs, fmt.Errorf("line %d: want \"key action\", \"key run command\", \"key tag text\" or \"prefix key action\", got %q", lineno, line))
			continue
		}
		key, action := fields[0], fields[1]
		var err error
		if action == "-" {
			err = target.Unbind(key)
		} else {
			err = target.Bind(key, action)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("line %d: %v", lineno, err))
			continue
		}
		n++
	}
	if err := sc.Err(); err != nil {
		errs = append(errs, err)
	}
	return n, errs
}

// loadKeysFile builds the direct and prefix keymaps from the defaults plus
// the keys file at path and installs them in global. A missing file is not an error. The
// number of applied lines and any per-line errors are returned for the
// caller to report.
func loadKeysFile(path string) (n int, errs []error) {
	km, pkm := DefaultKeymap(), DefaultPrefixKeymap()
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			errs = append(errs, err)
		}
		global.keymap, global.prefixKeymap = km, pkm
		return 0, errs
	}
	defer f.Close()
	n, errs = loadKeysText(km, pkm, f)
	global.keymap, global.prefixKeymap = km, pkm
	return n, errs
}

// bindingsDoc lists the direct and prefix bindings in keys file syntax.
func bindingsDoc() string {
	var sb strings.Builder
	sb.WriteString(global.keymap.String())
	for _, line := range strings.Split(strings.TrimRight(global.prefixKeymap.String(), "\n"), "\n") {
		if line != "" {
			sb.WriteString("prefix " + line + "\n")
		}
	}
	return sb.String()
}

// actionsDoc lists every action with its description, sorted by name.
func actionsDoc() string {
	names := make([]string, 0, len(actionTable))
	for name := range actionTable {
		names = append(names, name)
	}
	sort.Strings(names)
	var sb strings.Builder
	for _, name := range names {
		fmt.Fprintf(&sb, "%-14s %s\n", name, actionTable[name].Doc)
	}
	fmt.Fprintf(&sb, "%-14s %s\n", "run <command>", "run the command in the window, like button 2 on that text")
	fmt.Fprintf(&sb, "%-14s %s\n", "tag <text>", "type the text into the tag to complete, then Esc and ^E")
	return sb.String()
}

// keys implements the Keys command.
//
//	Keys          list the current bindings and the keys file path
//	Keys reload   re-read the keys file on top of the defaults
//	Keys actions  list the actions a key can be bound to
//	Keys file     print the keys file path
//
// It reaches keysCommand through a variable set in init so that the exec
// table, which lists Keys, does not statically refer to the action table:
// execute is itself an action, and that reference would make package
// initialization cyclic.
func keys(et *Text, seltext *Text, argt *Text, flag1, flag2 bool, arg string) {
	keysCommand(et, seltext, argt, flag1, flag2, arg)
}

var keysCommand func(et *Text, seltext *Text, argt *Text, flag1, flag2 bool, arg string)

func init() {
	keysCommand = keysCommandImpl
}

func keysCommandImpl(_ *Text, _ *Text, argt *Text, _, _ bool, arg string) {
	r, _ := getarg(argt, false, true)
	if r != "" {
		arg = r
	}
	arg = strings.TrimSpace(arg)
	path := keysFilePath()
	switch arg {
	case "":
		warning(nil, "Keys: bindings (file %s):\n%s", path, bindingsDoc())
	case "reload":
		n, errs := loadKeysFile(path)
		for _, err := range errs {
			warning(nil, "Keys: %s: %v\n", path, err)
		}
		warning(nil, "Keys: %s: %d binding(s) applied on top of defaults\n", path, n)
	case "actions":
		warning(nil, "Keys: actions:\n%s", actionsDoc())
	case "file":
		warning(nil, "%s\n", path)
	default:
		warning(nil, "Keys: unknown argument %q; use reload, actions or file\n", arg)
	}
}
