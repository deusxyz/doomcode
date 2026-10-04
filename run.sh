#!/bin/sh
# Run the locally built Edwood in its own namespace so it can coexist with
# a plan9port acme (both post the service "acme").
#
# devdraw: the plan9port C implementation. It is mature (fast drawing,
# cursors, mouse wheel, correct Backspace/Delete, key auto-repeat, Cmd keys);
# the Go devdraw from 9fans.net/go is not, see docs/03-keyboard-spec.md §7.
# Set DEVDRAW explicitly to try another one.
cd "$(dirname "$0")"
export PLAN9=${PLAN9:-$HOME/projects/plan9}
export PATH=$PATH:$PLAN9/bin
export NAMESPACE=${NAMESPACE:-/tmp/ns.edwood}
export DEVDRAW=${DEVDRAW:-$PLAN9/bin/devdraw}
mkdir -p "$NAMESPACE"
exec ./bin/edwood -f "$PLAN9/font/lucsans/euro.8.font" -F "$PLAN9/font/lucm/unicode.9.font" "$@"
