package model

import (
	"database/sql"
	"errors"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xo/dburl"
)

var databaseUrl string
var db *sql.DB

func init() {
	databaseUrl = os.Getenv("DATABASE_URL")
	var err error
	db, err = dburl.Open(databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
}

type Model struct {
	user User
	tx   *sql.Tx
	Err  error
}

func MakeModel(userID string, adminOnly bool) Model {
	if userID == "" {
		return Model{Err: errors.New("Not logged in")}
	}
	user := getUserByID(userID)
	if user == nil {
		return Model{Err: errors.New("No such user")}
	}
	model := Model{user: *user}
	model.tx, model.Err = db.Begin()
	return model
}

func (m *Model) Free() {
	m.Err = m.tx.Commit()
}

func (m *Model) User() User {
	return m.user
}
