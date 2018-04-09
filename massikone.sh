#!/bin/sh
set -eu
cd "$(dirname "$0")"
export GOPATH=$PWD
export PATH=$PWD/bin:$PATH
set -x
packr
go build
packr clean
