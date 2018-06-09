package model

import (
	"database/sql"
	"errors"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xo/dburl"
)

var db *sql.DB

type Model struct {
	user User
	tx   *sql.Tx
	Err  error
}

func getDB() *sql.DB {
	if db == nil {
		var err error
		db, err = dburl.Open(os.Getenv("DATABASE_URL"))
		if err != nil {
			log.Fatal(err)
		}
	}
	return db
}

func MakeModel(userID int64, adminOnly bool) Model {
	if userID == 0 {
		return Model{Err: errors.New("Not logged in")}
	}
	var m Model
	m.tx, m.Err = getDB().Begin()
	var user *User
	user, m.Err = m.getUserByID(userID)
	if user == nil && m.Err == nil {
		m.Err = errors.New("No such user")
	}
	m.user = *user
	return m
}

func (m *Model) Close() {
	if m.Err != nil {
		log.Print(m.Err)
		m.Err = m.tx.Rollback()
	} else {
		m.Err = m.tx.Commit()
	}
	if m.Err != nil {
		log.Print(m.Err)
	}
}

func (m *Model) User() User {
	return m.user
}
