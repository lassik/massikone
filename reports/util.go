package reports

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"golang.org/x/text/unicode/norm"
)

type GetWriter func(mimeType, filename string) (io.Writer, error)

type cell struct {
	text        string
	bold        bool
	rightAlign  bool
	width       int
	indentLevel int
}

type document struct {
	title     string
	filename  string
	orgName   string
	period    string
	printDate string
	headerRow []cell
	rows      [][]cell
}

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
	writer, err := getWriter("application/pdf",
		generateFilename(filename)+".pdf")
	check(err)
	check(pdf.Output(writer))
}

func writePdf(doc document, getWriter GetWriter) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 9)
	trFromUtf8 := pdf.UnicodeTranslatorFromDescriptor("")
	tr := func(s string) string {
		return trFromUtf8(norm.NFC.String(s))
	}
	const topMargin = 25
	const sideMargin = 10
	pdf.SetMargins(sideMargin, topMargin, sideMargin)
	pageWidth, _ := pdf.GetPageSize()
	pageWidth -= 2 * sideMargin
	pdf.SetHeaderFunc(func() {
		div3 := pageWidth / 3
		const height = 8.0
		pdf.SetY(5)
		pdf.SetX(sideMargin)
		pdf.SetFont("", "", 11)
		pdf.CellFormat(div3, height,
			tr(doc.orgName),
			"", 0, "L", false, 0, "")
		pdf.SetFont("", "B", 0)
		pdf.CellFormat(div3, height,
			tr(doc.title),
			"", 0, "C", false, 0, "")
		pdf.SetFont("", "", 0)
		pdf.CellFormat(div3, height,
			tr(fmt.Sprintf("Sivu %d", pdf.PageNo())),
			"", 1, "R", false, 0, "")
		pdf.SetX(sideMargin + div3)
		pdf.CellFormat(div3, height, tr(doc.period),
			"", 0, "C", false, 0, "")
		pdf.CellFormat(div3, height, tr(doc.printDate),
			"", 1, "R", false, 0, "")
		headerTotalWidth := float64(0)
		for _, thisCell := range doc.headerRow {
			headerTotalWidth += float64(thisCell.width)
		}
		if headerTotalWidth < 1 {
			panic("headerTotalWidth < 1")
		}
		multiplier := pageWidth / headerTotalWidth
		pdf.SetX(sideMargin)
		pdf.SetFont("", "B", 0)
		for _, thisCell := range doc.headerRow {
			const height = 7.0
			width := multiplier * float64(thisCell.width)
			align := "L"
			if thisCell.rightAlign {
				align = "R"
			}
			pdf.CellFormat(width, height, tr(thisCell.text),
				"", 0, align, false, 0, "")
		}
		pdf.Ln(-1)
	})
	pdf.AddPage()
	for _, thisRow := range doc.rows {
		totalWidth := float64(0)
		for _, thisCell := range thisRow {
			totalWidth += float64(thisCell.width)
		}
		if totalWidth < 1 {
			panic("totalWidth < 1")
		}
		multiplier := pageWidth / totalWidth
		pdf.SetX(sideMargin)
		for _, thisCell := range thisRow {
			height := 5.0
			indentWidth := float64(thisCell.indentLevel) * 4
			if indentWidth > 0 {
				pdf.CellFormat(indentWidth, height, "",
					"", 0, "", false, 0, "")
			}
			if thisCell.bold {
				pdf.SetFont("", "B", 0)
			} else {
				pdf.SetFont("", "", 0)
			}
			width := multiplier*float64(thisCell.width) - indentWidth
			align := "L"
			if thisCell.rightAlign {
				align = "R"
			}
			pdf.CellFormat(width, height, tr(thisCell.text),
				"", 0, align, false, 0, "")
		}
		pdf.Ln(-1)
	}
	writer, err := getWriter("application/pdf",
		generateFilename(doc.filename)+".pdf")
	check(err)
	check(pdf.Output(writer))
}
