package main

import "github.com/lassik/airfreight/packer"

func main() {
	packer.Package("main").Map("static", "static", "static-external").
		WriteFile("static.go")
	packer.Package("main").Map("templates", "templates").
		WriteFile("templates.go")
}
