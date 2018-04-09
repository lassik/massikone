package reports

import (
	"io"
	"log"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

type GetWriter func(mimeType, filename string) (io.Writer, error)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
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

func generateFilename(document string) string {
	year := "2018" // TODO
	//prefs := modelGetPreferences()
	//orgShortName := prefs["org_short_name"]
	orgShortName := "Testi"
	return Slug(orgShortName + "-" + year + "-" + document)
}

func blankPdf(getWriter GetWriter, filename string) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Hello, world")
	writer, err := getWriter("application/pdf", generateFilename(filename))
	check(err)
	err = pdf.Output(writer)
	check(err)
}
