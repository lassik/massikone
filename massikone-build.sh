#!/bin/sh
set -eux
cd "$(dirname "$0")"
export GOPATH=$PWD/.go
export PATH=$GOPATH/bin:$PATH
export CGO_ENABLED=1 # Required for sqlite3
set -x
! which cloc >/dev/null 2>&1 || cloc massikone.go model reports static views
packr
go build -o massikone
packr clean
gofmt -l *.go model reports
