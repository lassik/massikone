package reports

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"path"

	"github.com/jung-kurt/gofpdf"

	"../model"
)

type GetWriter func(mimeType, filename string) (io.Writer, error)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
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

func IncomeStatementPdf(getWriter GetWriter) {
	blankPdf(getWriter, "tuloslaskelma")
}

func IncomeStatementDetailedPdf(getWriter GetWriter) {
	blankPdf(getWriter, "tuloslaskelma erittelyin")
}

func BalanceSheetPdf(getWriter GetWriter) {
	blankPdf(getWriter, "tase")
}

func BalanceSheetDetailedPdf(getWriter GetWriter) {
	blankPdf(getWriter, "tase erittelyin")
}

func GeneralJournalPdf(getWriter GetWriter) {
	blankPdf(getWriter, "päiväkirja")
}

func GeneralLedgerPdf(getWriter GetWriter) {
	blankPdf(getWriter, "pääkirja")
}

func ChartOfAccountsPdf(getWriter GetWriter) {
	blankPdf(getWriter, "tilikartta")
}

func addBillImagesToZip(getWriter GetWriter) {
	images, missing := model.GetBillsForImages()
	for _, image := range images {
		if image["image_id"] != nil {
			w, err := getWriter(
				"image/"+path.Ext(image["image_id"].(string)),
				fmt.Sprintf("tositteet/tosite-%03d-%d-%s%s",
					image["bill_id"].(string),
					image["bill_image_num"].(string),
					Slug(image["description"].(string)),
					path.Ext(image["image_id"].(string))))
			if err != nil {
				log.Fatal(err)
			}
			_, err = w.Write(image["image_data"].([]byte))
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	if len(missing) > 0 {
		w, err := getWriter("text/plain", "tositteet/puuttuvat.txt")
		check(err)
		for _, billId := range missing {
			fmt.Fprintf(w, "#%d\r\n", billId)
		}
	}
}

func FullStatementZip(getWriter GetWriter) {
	zipFilename := generateFilename("tilinpaatos")
	zipBasename := path.Base(zipFilename)
	outerWriter, err := getWriter("application/zip", zipFilename)
	if err != nil {
		log.Fatal(err)
	}
	//zipBuf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(outerWriter) //zip.NewWriter(zipBuf)
	writeToZip := func(_, filename string) (io.Writer, error) {
		return zipWriter.Create(zipBasename + "/" + filename)
	}
	GeneralJournalPdf(writeToZip)
	ChartOfAccountsPdf(writeToZip)
	addBillImagesToZip(writeToZip)
	err = zipWriter.Close()
	if err != nil {
		log.Fatal(err)
	}
	//io.Copy(outerWriter, zipBuf)

}
