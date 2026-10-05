package theme

import (
	"strings"
	"testing"

	"github.com/rjkroege/edwood/draw"
)

func TestParseColor(t *testing.T) {
	for s, want := range map[string]draw.Color{
		"#000000": 0x000000FF, "#ffffff": 0xFFFFFFFF, "#1f3a93": 0x1F3A93FF,
		"#abc": 0xAABBCCFF, "#11223344": 0x11223344,
	} {
		got, err := ParseColor(s)
		if err != nil || got != want {
			t.Errorf("ParseColor(%q) = %#x,%v; want %#x", s, uint32(got), err, uint32(want))
		}
	}
	for _, bad := range []string{"", "red", "#", "#12", "#12345", "#gggggg", "123456"} {
		if _, err := ParseColor(bad); err == nil {
			t.Errorf("ParseColor(%q) succeeded", bad)
		}
	}
	if ColorString(0x1F3A93FF) != "#1f3a93" || ColorString(0x11223344) != "#11223344" {
		t.Errorf("ColorString: %s %s", ColorString(0x1F3A93FF), ColorString(0x11223344))
	}
}

func TestParseTheme(t *testing.T) {
	tf, errs := ParseTheme(strings.NewReader(`
# a comment
palette vampira
style keyword fg=#123456
style error bg=#ffd6d6 underline line=#c02020   # trailing comment
style comment -
style keyword bg=#eeeeee          # adds to the earlier keyword line
text.back #ffffea mix #ffffff
tag.text #000
bogus line here
style broken fg=red
ui.but2 #aa0000
nope.slot #000000
`))
	if len(errs) != 3 {
		t.Fatalf("errs = %v; want 3", errs)
	}
	for i, want := range []string{"line 10", "line 11", "line 13"} {
		if !strings.Contains(errs[i].Error(), want) {
			t.Errorf("errs[%d] = %v; want %s", i, errs[i], want)
		}
	}
	if tf.Palette != "vampira" {
		t.Errorf("palette %q", tf.Palette)
	}
	k := tf.Styles["keyword"]
	if k == nil || k.Fg.Color != 0x123456FF || k.Bg.Color != 0xEEEEEEFF {
		t.Errorf("keyword = %+v", k)
	}
	e := tf.Styles["error"]
	if e == nil || e.Bg.Color != 0xFFD6D6FF || !e.Underline || e.Line.Color != 0xC02020FF {
		t.Errorf("error = %+v", e)
	}
	if c, ok := tf.Styles["comment"]; !ok || c != nil {
		t.Errorf("comment should be removed: %v %v", c, ok)
	}
	if _, ok := tf.Styles["broken"]; ok {
		t.Error("broken style was kept")
	}
	if s := tf.Slots["text.back"]; s.Color != 0xFFFFEAFF || s.Mix != 0xFFFFFFFF {
		t.Errorf("text.back = %+v", s)
	}
	if s := tf.Slots["tag.text"]; s.Color != 0x000000FF || s.Mix != 0 {
		t.Errorf("tag.text = %+v", s)
	}
	if s := tf.Slots["ui.but2"]; s.Color != 0xAA0000FF {
		t.Errorf("ui.but2 = %+v", s)
	}
}

func TestThemeApply(t *testing.T) {
	p := Light // copy
	tf, errs := ParseTheme(strings.NewReader("style keyword fg=#123456\nstyle comment -\nstyle brand fg=#ff00ff\ntext.back #101010\nui.but3 #00ff00\n"))
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	tf.Apply(&p)
	if p.Text.Back.Color != 0x101010FF || p.Text.Back.Mix != 0 {
		t.Errorf("text.back = %+v", p.Text.Back)
	}
	if p.Ui.But3.Color != 0x00FF00FF {
		t.Errorf("ui.but3 = %+v", p.Ui.But3)
	}
	if s, ok := p.Styles.Resolve("keyword"); !ok || s.Fg.Color != 0x123456FF {
		t.Errorf("keyword = %+v %v", s, ok)
	}
	if _, ok := p.Styles.Resolve("comment"); ok {
		t.Error("comment still defined after removal")
	}
	if s, ok := p.Styles.Resolve("brand"); !ok || s.Fg.Color != 0xFF00FFFF {
		t.Errorf("brand = %+v %v", s, ok)
	}
	// The built-in palette is untouched.
	if _, ok := Light.Styles.Resolve("comment"); !ok {
		t.Error("Apply modified the built-in Light palette")
	}
	if Light.Text.Back.Color == 0x101010FF {
		t.Error("Apply modified Light.Text.Back")
	}
	doc := p.StylesDoc()
	if !strings.Contains(doc, "style keyword      fg=#123456") || !strings.Contains(doc, "style error        bg=#ffd6d6 underline line=#c02020") {
		t.Errorf("StylesDoc:\n%s", doc)
	}
}
