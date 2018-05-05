package model

import (
	"log"
	"strconv"
	"time"
)

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
