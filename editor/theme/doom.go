package theme

// doomcode's own palettes, chosen by the style guide in
// docs/10-theme-style-guide.md: soft backgrounds instead of pure white or
// black, body text near 10:1 against them, comments at least 4.5:1, and
// syntax colours of similar weight that differ mainly in hue.
//
// doom-light softens Acme's palette: the same pale yellow body and
// blue-green tags, warmer and less saturated, and dark grey text instead
// of black. doom-dark follows doom-one from doom-themes (Henrik Lissner,
// MIT, https://github.com/doomemacs/themes), the theme of the Doom Emacs
// website, with comments brightened to meet the contrast floor.

// DoomLight is the default palette.
var DoomLight = Palette{
	Tag: FramePalette{
		Back:  solid(0xE4F1EFFF),
		High:  solid(0xC6DEDAFF),
		Bord:  solid(0x8C8AC4FF),
		Text:  solid(0x3A3A34FF),
		HText: solid(0x3A3A34FF),
		Tick:  solid(0x3A3A34FF),
	},
	Text: FramePalette{
		Back:  solid(0xFBF8E8FF),
		High:  solid(0xECE4B0FF),
		Bord:  solid(0xB7BA86FF),
		Text:  solid(0x3A3A34FF),
		HText: solid(0x3A3A34FF),
		Tick:  solid(0x3A3A34FF),
	},
	Ui: UiPalette{
		ModButton: solid(0x3B5BA5FF),
		ColButton: solid(0x8C8AC4FF),
		But2:      solid(0xB5524AFF),
		But3:      solid(0x4E8A4EFF),
	},
	Styles:    doomLightStyles,
	VarFont:   "/mnt/font/Avenir-Book/14a/font",
	FixedFont: "/mnt/font/Menlo-Regular/13a/font",
}

var doomLightStyles = Styles{
	"comment":     fg(0x646F61FF),
	"keyword":     fg(0x35539AFF),
	"string":      fg(0x94552BFF),
	"number":      fg(0x7A4C98FF),
	"constant":    fg(0x86497FFF),
	"type":        fg(0x2E716FFF),
	"function":    fg(0x3F5C94FF),
	"preproc":     fg(0x7D6420FF),
	"heading":     fg(0x35539AFF),
	"emphasis":    fg(0x5C4A86FF),
	"link":        StyleSpec{Fg: solid(0x3468B0FF), Underline: true},
	"error":       bgLine(0xF6DCD8FF, 0xB5524AFF),
	"warning":     bgLine(0xF4EBC8FF, 0xA07A20FF),
	"info":        bgLine(0xDCE7F3FF, 0x3F6AA8FF),
	"hint":        bgLine(0xE0EDDAFF, 0x4E8A4EFF),
	"match":       bg(0xF0DE9AFF),
	"diff.add":    bg(0xDFF0D8FF),
	"diff.del":    bg(0xF6DCD8FF),
	"diff.change": bg(0xF4EBC8FF),
}

// doom-one colours (doom-themes, MIT).
const (
	doomBg      = 0x282C34FF
	doomBgAlt   = 0x21242BFF
	doomBase4   = 0x3F444AFF
	doomRegion  = 0x42444AFF
	doomFg      = 0xBBC2CFFF
	doomRed     = 0xFF6C6BFF
	doomOrange  = 0xDA8548FF
	doomGreen   = 0x98BE65FF
	doomTeal    = 0x4DB5BDFF
	doomYellow  = 0xECBE7BFF
	doomBlue    = 0x51AFEFFF
	doomMagenta = 0xC678DDFF
	doomViolet  = 0xA9A1E1FF
	doomComment = 0x959CA9FF // doom-one has 0x5B6268, below 4.5:1
)

// DoomDark is the dark palette after doom-one.
var DoomDark = Palette{
	Tag: FramePalette{
		Back:  solid(doomBgAlt),
		High:  solid(doomBase4),
		Bord:  solid(doomBase4),
		Text:  solid(doomFg),
		HText: solid(doomFg),
		Tick:  solid(doomBlue),
	},
	Text: FramePalette{
		Back:  solid(doomBg),
		High:  solid(doomRegion),
		Bord:  solid(doomBase4),
		Text:  solid(doomFg),
		HText: solid(doomFg),
		Tick:  solid(doomBlue),
	},
	Ui: UiPalette{
		ModButton: solid(doomBlue),
		ColButton: solid(0x5B6268FF),
		But2:      solid(doomRed),
		But3:      solid(doomGreen),
	},
	Styles:    doomDarkStyles,
	VarFont:   "/mnt/font/HelveticaNeue/14a/font",
	FixedFont: "/mnt/font/Menlo-Regular/13a/font",
}

var doomDarkStyles = Styles{
	"comment":     fg(doomComment),
	"keyword":     fg(doomBlue),
	"string":      fg(doomGreen),
	"number":      fg(doomOrange),
	"constant":    fg(doomViolet),
	"type":        fg(doomYellow),
	"function":    fg(doomMagenta),
	"preproc":     fg(doomTeal),
	"heading":     fg(doomBlue),
	"emphasis":    fg(doomViolet),
	"link":        StyleSpec{Fg: solid(doomBlue), Underline: true},
	"error":       bgLine(0x3A2A2FFF, doomRed),
	"warning":     bgLine(0x34322BFF, doomYellow),
	"info":        bgLine(0x24303DFF, doomBlue),
	"hint":        bgLine(0x29342CFF, doomGreen),
	"match":       bg(0x5A5030FF),
	"diff.add":    bg(0x29342CFF),
	"diff.del":    bg(0x3A2A2FFF),
	"diff.change": bg(0x34322BFF),
}
