package model

import (
	"database/sql"
	"errors"
	"log"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/mattn/go-sqlite3"
	"github.com/xo/dburl"
)

var db *sql.DB

type Model struct {
	user User
	tx   *sql.Tx
	Err  error
}

func getVersion(tx *sql.Tx) int {
	var version int
	sq.Select("version").From("version").RunWith(tx).QueryRow().Scan(&version)
	return version
}

func migrate(tx *sql.Tx) {
	migs := []string{"/0to1.sql"}
	maxVersion := len(migs)
	oldVersion := getVersion(tx)
	log.Printf("Tietokannan versio: %d", oldVersion)
	if oldVersion > maxVersion {
		log.Fatal("Tietokanta vaatii uudemman version Massikoneesta")
	}
	for m := oldVersion; m < maxVersion; m++ {
		log.Printf("Muunnetaan tietokanta uudempaan muotoon (%s)", migs[m])
		if _, err := tx.Exec(migrations[migs[m]].Contents); err != nil {
			log.Fatal(err)
		}
	}
}

func Initialize(databaseURL string) {
	if db != nil {
		return
	}
	var tx *sql.Tx
	var err error
	log.Printf("Tietokanta: %s", databaseURL)
	if db, err = dburl.Open(databaseURL); err != nil {
		log.Fatal(err)
	}
	if tx, err = db.Begin(); err != nil {
		log.Fatal(err)
	}
	migrate(tx)
	if err = tx.Commit(); err != nil {
		log.Fatal(err)
	}
}

func getDB() *sql.DB {
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
	if adminOnly && (user == nil || !user.IsAdmin) {
		m.Err = errors.New("Forbidden")
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
