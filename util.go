package main

import (
	"strings"
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

func Shorten(str string) string {
	str = strings.SplitN(str, "\n", 2)[0]
	//.gsub(`\s+`, ' ').strip.slice(0:50)
	return str
}

func Slug(str string) string {
	str = strings.SplitN(str, "\n", 2)[0]
	str = strings.ToLower(str)
	//str = str.gsub(`\s+`, '-').gsub(`[^\w-]`, "")
	//str = str.gsub(`--+`, '-').gsub(`^-`, "")
	str = Shorten(str)
	//str = str.gsub(`-$`, "")
	return str
}
