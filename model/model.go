package model

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xo/dburl"
)

var databaseUrl = os.Getenv("DATABASE_URL")
var db *sql.DB

func init() {
	var err error
	db, err = dburl.Open(databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
}
