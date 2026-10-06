#!/bin/sh
# Build Edwood into bin/. The devdraw used at run time is plan9port's
# ($PLAN9/bin/devdraw, see run.sh); pass --go-devdraw to also build the Go
# devdraw from the 9fans-go fork clone ($NINEFANS_GO) for experiments.
set -e
cd "$(dirname "$0")"
mkdir -p bin
(cd edwood && go build -o ../bin/edwood .)
go build -o bin/Syn ./cmd/Syn
go build -o bin/Diag ./cmd/Diag
if [ "$1" = "--go-devdraw" ]; then
	NINEFANS_GO=${NINEFANS_GO:-$HOME/projects/9fans/go}
	(cd "$NINEFANS_GO" && go build -o "$OLDPWD/bin/devdraw" ./cmd/devdraw 2>&1 | grep -v -e deprecated -e '^\s' -e 'note:' || true)
fi
/bin/ls -la bin/
