package model

import (
	"database/sql"
	"errors"
	"log"
	"os"

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

func EnsureInitializedDB() {
	if db != nil {
		return
	}
	var err error
	log.Printf("Tietokanta: %s", os.Getenv("DATABASE_URL"))
	db, err = dburl.Open(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	migrate(tx)
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
}

func getDB() *sql.DB {
	EnsureInitializedDB()
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
