package main

import (
	"strings"
)

// Keys bound to commands. In the keys file "F12 run L def" binds F12 to
// running the command "L def" in the focused window, exactly as if the
// text "L def" had been clicked with button 2 there: built-in commands
// (Put, Undo, Edit ...) run inside Edwood, anything else is started with
// $winid set to the window, in the window's directory, with output in
// +Errors. The selection is left alone, so tools such as acme-lsp's L read
// the cursor position from the window's dot.
//
// See docs/03-keyboard-spec.md section 5.4 in the justcode project.

// runPrefix starts an action name that runs a command; tagPrefix one
// that types a command into the tag for the user to complete, for
// commands that need an argument (tag L rn: type the new name, Esc, ^E),
// the way find and goto do.
const (
	runPrefix = "run "
	tagPrefix = "tag "
)

// commandAction returns the action for "run <command>" or "tag <text>",
// or nil when name is neither.
func commandAction(name string) *Action {
	if a := runAction(name); a != nil {
		return a
	}
	if !strings.HasPrefix(name, tagPrefix) {
		return nil
	}
	text := strings.Join(strings.Fields(name[len(tagPrefix):]), " ")
	if text == "" {
		return nil
	}
	return &Action{
		Name: tagPrefix + text,
		Doc:  "type \"" + text + " \" into the tag to complete, then Esc and ^E",
		Fn:   func(t *Text) { t.typeIntoTag(text + " ") },
	}
}

// runAction returns the action for "run <command>", or nil when name is
// not of that form or the command is empty.
func runAction(name string) *Action {
	if !strings.HasPrefix(name, runPrefix) {
		return nil
	}
	cmd := strings.Join(strings.Fields(name[len(runPrefix):]), " ")
	if cmd == "" {
		return nil
	}
	return &Action{
		Name: runPrefix + cmd,
		Doc:  "run " + cmd + " in the window, like button 2 on that text",
		Fn:   func(t *Text) { runKeyCommand(t, cmd) },
	}
}

// runKeyCommand is set in init: calling executeCommand directly from the
// key tables would make package initialization cyclic, as for Keys.
var runKeyCommand func(t *Text, cmd string)

func init() {
	runKeyCommand = executeCommand
}

// executeCommand runs cmd as button 2 would run that text in t's window.
func executeCommand(t *Text, cmd string) {
	t.commitAll()
	if e := lookup(cmd, globalexectab); e != nil {
		if e.mark && global.seltext != nil && global.seltext.what == Body {
			global.seq++
			global.seltext.w.body.file.Mark(global.seq)
		}
		arg := ""
		if words := wsre.Split(cmd, 2); len(words) > 1 {
			arg = strings.TrimLeft(words[1], " \t\n")
		}
		seltext := global.seltext
		if seltext == nil {
			seltext = t
		}
		e.fn(t, seltext, nil, e.flag1, e.flag2, arg)
		return
	}
	if t.w != nil {
		t.w.ref.Inc()
	}
	run(t.w, cmd, t.DirName(""), true, "", "", false)
}
