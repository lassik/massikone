package reports

import (
	"io"
	"log"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"golang.org/x/text/unicode/norm"
)

type GetWriter func(mimeType, filename string) (io.Writer, error)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func shorten(str string) string {
	str = strings.SplitN(str, "\n", 2)[0]
	//.gsub(`\s+`, ' ').strip.slice(0:50)
	return str
}

func slug(str string) string {
	str = strings.SplitN(str, "\n", 2)[0]
	str = strings.ToLower(str)
	//str = str.gsub(`\s+`, '-').gsub(`[^\w-]`, "")
	//str = str.gsub(`--+`, '-').gsub(`^-`, "")
	str = shorten(str)
	//str = str.gsub(`-$`, "")
	return str
}

func generateFilename(document string) string {
	year := "2018" // TODO
	//settings := modelGetSettings()
	//orgShortName := settings["org_short_name"]
	orgShortName := "Testi"
	return norm.NFC.String(slug(orgShortName + "-" + year + "-" + document))
}

func blankPdf(getWriter GetWriter, filename string) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Hello, world")
	writer, err := getWriter("application/pdf",
		generateFilename(filename)+".pdf")
	check(err)
	err = pdf.Output(writer)
	check(err)
}
