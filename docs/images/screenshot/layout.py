# layout.py VARFONT FIXFONT BANNER > layout.dump
# Writes the Dump file for the README screenshot: three columns, the
# repository listing, the banner and README, two Go files. The repository
# is expected at /tmp/doomcode (short path, no home directory in tags).
import json, sys
import os
P = os.path.join(os.environ.get("PLAN9", os.path.expanduser("~/projects/plan9")), "font")
var = (sys.argv[1] if len(sys.argv) > 1 else "") or P + "/lucsans/euro.8.font"
fix = (sys.argv[2] if len(sys.argv) > 2 else "") or P + "/lucm/unicode.9.font"
R = "/tmp/doomcode/"
banner = open(sys.argv[3]).read()
rd = open(R + "README.md").read()
readme_q = len(rd[:rd.index("# Introduction")])
def tag(s): return {"Buffer": s, "Q0": len(s), "Q1": len(s)}
def win(col, pos, path, font, kind=0, body=None, extra=" Del Snarf | Look Edit "):
    w = {"Type": kind, "Column": col, "Position": pos, "Font": font, "Tag": tag(path + extra), "Body": {"Q0": 0, "Q1": 0}}
    if body is not None:
        w["Body"]["Buffer"] = body
    return w
col_tag = tag("New Cut Paste Snarf Sort Zerox Delcol ")
d = {
    "Version": 1, "CurrentDir": R, "VarFont": var, "FixedFont": fix,
    "RowTag": tag("Newcol Kill Putall Dump Exit "),
    "Columns": [
        {"Position": 0, "Tag": col_tag},
        {"Position": 19.5, "Tag": col_tag},
        {"Position": 57, "Tag": col_tag},
    ],
    "Windows": [
        win(0, 0, R, var, extra=" Del Snarf Get | Look Edit "),
        win(0, 20.5, R + "docs/", var, extra=" Del Snarf Get | Look Edit "),
        win(1, 0, R + "+DOOM", fix, kind=1, body=banner, extra=" Del Snarf | Look Edit "),
        dict(win(1, 40, R + "README.md", var), Body={"Q0": readme_q, "Q1": readme_q}),
        win(2, 0, R + "editor/keys.go", fix),
        win(2, 52, R + "cmd/Syn/doc.go", fix),
    ],
}
json.dump(d, sys.stdout, indent=1)
