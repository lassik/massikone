package model

import (
	"time"
)

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
