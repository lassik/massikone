package model

import (
	"database/sql"
	"log"
	"strconv"
	"time"

	sq "github.com/Masterminds/squirrel"
)

func (m *Model) isErr(err error) bool {
        if m.Err == nil {
                m.Err = err
        }
        return err != nil
}

func parsePositiveInt(what, s string) int {
	var val64 int64
	val64, err := strconv.ParseInt(s, 10, 32)
	var val int
	val = int(val64)
	if err != nil || val < 1 {
		log.Printf("Invalid %s: %q", what, s)
		return -1
	}
	return val
}

func FiFromIsoDate(str string) string {
	if str == "" {
		return ""
	}
	date, err := time.Parse("2006-01-02", str)
	if err != nil {
		return ""
	}
	return date.Format("2.1.2006")
}

func (m *Model) getIntFromDb(q sq.SelectBuilder) string {
	var val sql.NullString
	err := q.RunWith(m.tx).Limit(1).QueryRow().Scan(&val)
	if err != sql.ErrNoRows {
		m.isErr(err)
	}
	if err != nil {
		return ""
	}
	return val.String
}
