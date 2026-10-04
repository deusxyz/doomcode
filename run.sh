#!/bin/sh
# Run the locally built Edwood with the forked devdraw, in its own namespace
# so it can coexist with a plan9port acme (both post the service "acme").
cd "$(dirname "$0")"
export PLAN9=${PLAN9:-$HOME/projects/plan9}
export PATH=$PATH:$PLAN9/bin
export NAMESPACE=${NAMESPACE:-/tmp/ns.edwood}
export DEVDRAW=$PWD/bin/devdraw
mkdir -p "$NAMESPACE"
exec ./bin/edwood -f "$PLAN9/font/lucsans/euro.8.font" -F "$PLAN9/font/lucm/unicode.9.font" "$@"
