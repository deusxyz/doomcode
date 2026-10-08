#!/bin/sh
# Run the locally built doomcode in its own namespace so it can coexist with
# a plan9port acme (both post the service "acme").
#
# devdraw: the plan9port C implementation. It is mature (fast drawing,
# cursors, mouse wheel, correct Backspace/Delete, key auto-repeat, Cmd keys);
# the Go devdraw from 9fans.net/go is not, see docs/03-keyboard-spec.md §7.
# Set DEVDRAW explicitly to try another one.
cd "$(dirname "$0")"
export PLAN9=${PLAN9:-$HOME/projects/plan9}
export PATH=$PWD/bin:$HOME/go/bin:$PATH:$PLAN9/bin # bin/ first: Syn is run from a tag
export NAMESPACE=${NAMESPACE:-/tmp/ns.doomcode}
export DEVDRAW=${DEVDRAW:-$PLAN9/bin/devdraw}
mkdir -p "$NAMESPACE"
# Fonts come from the config file or the palette (see docs/configuration.md);
# pass -f and -F here to force them.
exec ./bin/doomcode "$@"
