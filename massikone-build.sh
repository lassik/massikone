#!/bin/sh
set -eux
cd "$(dirname "$0")"
export GOPATH=$PWD/.go
export PATH=$GOPATH/bin:$PATH
set -x
! which cloc >/dev/null 2>&1 || cloc *.go model reports views public
packr
go build -o massikone
packr clean
gofmt -l *.go model reports
