package main

import (
	"time"
)

func fi_from_iso_date(str string) string {
	if str == "" {
		return ""
	}
	date, err := time.Parse("2006-01-02", str)
	if err != nil {
		return ""
	}
	return date.Format("2.1.2006")
}
