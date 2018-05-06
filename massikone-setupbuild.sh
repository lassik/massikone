#!/bin/sh
set -eux
go get -u github.com/disintegration/imaging
go get -u github.com/gobuffalo/packr
go get -u github.com/gobuffalo/packr/...
go get -u github.com/gorilla/handlers
go get -u github.com/gorilla/mux
go get -u github.com/gorilla/sessions
go get -u github.com/hoisie/mustache
go get -u github.com/jung-kurt/gofpdf
go get -u github.com/markbates/goth
go get -u github.com/Masterminds/squirrel
go get -u github.com/mattn/go-sqlite3
go get -u github.com/xo/dburl
