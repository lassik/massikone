#!/bin/sh
set -eu
cd "$(dirname "$0")"
export GOPATH=$PWD
export PATH=$PWD/bin:$PATH
set -x
packr
go build image.go massikone.go model.go reports.go util.go
packr clean
