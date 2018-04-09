package reports

import (
	"archive/zip"
	"io"
	"log"
	"path"
)

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
