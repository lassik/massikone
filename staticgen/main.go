package main

import "github.com/lassik/airfreight/packer"

func main() {
	packer.Package("main").
		Map("static", "static", "static-external").
		Map("templates", "templates").
		WriteFile("static.go")
}
