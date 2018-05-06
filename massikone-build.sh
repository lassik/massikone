#!/bin/sh
set -eu
cd "$(dirname "$0")"
export GOPATH=$PWD
export PATH=$PWD/bin:$PATH
set -x
gofmt -l *.go model reports
! which cloc >/dev/null 2>&1 || cloc *.go model reports views public
packr
go build -o massikone
packr clean