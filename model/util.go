package model

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
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

func isoFromFiDate(str string) string {
	if str == "" {
		return ""
	}
	date, err := time.Parse("2.1.2006", str)
	if err != nil {
		return ""
	}
	return date.Format("2006-01-02")
}

func fiFromISODate(str string) string {
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

func amountFromCents(cents int64) string {
	if cents <= 0 {
		return ""
	}
	euros := cents / 100
	cents = cents % 100
	return fmt.Sprintf("%d,%02d", euros, cents)
}

func centsFromAmount(amount string) (int64, error) {
	amount = regexp.MustCompile(`\s+`).ReplaceAllString(amount, "")
	if amount == "" {
		return 0, nil
	}
	ms := regexp.MustCompile(`^(\d+)(,(\d\d))?$`).FindStringSubmatch(amount)
	if ms == nil {
		return 0, fmt.Errorf("Invalid amount: %q", amount)
	}
	euros, err := strconv.Atoi(ms[1])
	if err != nil {
		return 0, err
	}
	cents := 0
	if ms[3] != "" {
		cents, err = strconv.Atoi(ms[3])
		if err != nil {
			return 0, err
		}
	}
	cents = (euros * 100) + cents
	return int64(cents), nil
}
