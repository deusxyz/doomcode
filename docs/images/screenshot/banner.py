D = ["______ ", "|  _  \\", "| | | |", "| | | |", "| |/ / ", "|___/  "]
O = [" _____ ", "|  _  |", "| | | |", "| | | |", "\\ \\_/ /", " \\___/ "]
M = ["___  ___", "|  \\/  |", "| .  . |", "| |\\/| |", "| |  | |", "\\_|  |_/"]
c = ["      ", "      ", "  ___ ", " / __|", "| (__ ", " \\___|"]
o = ["       ", "       ", "  ___  ", " / _ \\ ", "| (_) |", " \\___/ "]
d = ["     _ ", "    | |", "  __| |", " / _` |", "| (_| |", " \\__,_|"]
e = ["      ", "      ", "  ___ ", " / _ \\", "|  __/", " \\___|"]
rows = []
for i in range(6):
    rows.append("  " + D[i] + O[i] + O[i] + M[i] + "   " + c[i] + o[i] + d[i] + e[i])
text = "\n".join(r.rstrip() for r in rows)
text += """

  A minimalist, comfortable and fast editor for developers.
  The soul of Acme, the comfort of a modern editor.

        "Rip and tear, until it is done."

  repo      https://github.com/deusxyz/doomcode
  authors   Igor Kozlitin, with Claude (Anthropic)
  license   MIT

  Ctrl-B o  next window     F12     definition
  Ctrl-/    comment         Ctrl-S  format and Put
"""
print(text)
