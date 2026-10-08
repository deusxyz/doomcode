#!/bin/sh
# Retake docs/images/screenshot.png: clone the repository to /tmp/doomcode,
# load the layout in a separate namespace with Syn running, and capture
# the window. macOS only (screencapture, swiftc).
#
#	docs/images/screenshot/shoot.sh [palette] [size]
set -e
D=$(cd "$(dirname "$0")" && pwd)
R=$(cd "$D/../../.." && pwd)
PALETTE=${1:-vampira}
SIZE=${2:-1600x1000}
W=$(mktemp -d)
export PLAN9=${PLAN9:-$HOME/projects/plan9}
export PATH=$R/bin:$HOME/go/bin:/usr/bin:/bin:/usr/sbin:/sbin:$PLAN9/bin
export NAMESPACE=$W/ns DEVDRAW=$PLAN9/bin/devdraw DEVDRAW_MODKEYS=1
mkdir -p "$NAMESPACE"

rm -rf /tmp/doomcode && git clone -q "$R" /tmp/doomcode
swiftc -O -o "$W/winlist" "$D/winlist.swift"
python3 "$D/banner.py" > "$W/banner.txt"
python3 "$D/layout.py" "" "" "$W/banner.txt" > "$W/layout.dump"

cd /tmp/doomcode
"$R/bin/doomcode" -l "$W/layout.dump" -W "$SIZE" -palette "$PALETTE" > "$W/doomcode.log" 2>&1 &
ed=$!
sleep 2
"$R/bin/Syn" > "$W/syn.log" 2>&1 &
syn=$!
sleep 2
# The banner is an unsaved window: mark it clean so its tag has no Put.
b=$(9p read acme/index | awk '$6 ~ /\+DOOM$/ {print $1}')
[ -n "$b" ] && echo clean | 9p write "acme/$b/ctl"
sleep 1
id=$("$W/winlist" | awk -F'\t' '$4=="doomcode"{print $1}' | tail -1)
screencapture -x -o -l"$id" "$R/docs/images/screenshot.png"
kill $syn $ed 2>/dev/null || true
echo "wrote $R/docs/images/screenshot.png"
