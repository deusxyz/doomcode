#!/bin/sh
# Build Edwood and the forked devdraw into bin/.
#
# Edwood:  ./edwood (fork of rjkroege/edwood, remote origin = deusxyz/edwood)
# devdraw: cmd/devdraw in the 9fans.net/go fork (deusxyz/9fans-go), branch
#          devdraw/keys, cloned at $NINEFANS_GO (default ~/projects/9fans/go).
set -e
cd "$(dirname "$0")"
NINEFANS_GO=${NINEFANS_GO:-$HOME/projects/9fans/go}
mkdir -p bin
(cd edwood && go build -o ../bin/edwood .)
(cd "$NINEFANS_GO" && go build -o "$OLDPWD/bin/devdraw" ./cmd/devdraw 2>&1 | grep -v -e 'deprecated' -e '^\s' -e 'note:' || true)
ls -la bin/edwood bin/devdraw
