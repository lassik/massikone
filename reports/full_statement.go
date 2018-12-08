package reports

import (
	"archive/zip"
	"io"
	"log"
	"path"

	"github.com/lassik/massikone/model"
)

func FullStatementZip(m *model.Model, getWriter GetWriter) {
	zipFilename := generateFilename("tilinpäätos")
	zipBasename := path.Base(zipFilename)
	outerWriter, err := getWriter("application/zip", zipFilename+".zip")
	if err != nil {
		log.Fatal(err)
	}
	//zipBuf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(outerWriter) //zip.NewWriter(zipBuf)
	writeToZip := func(_, filename string) (io.Writer, error) {
		return zipWriter.Create(zipBasename + "/" + filename)
	}
	GeneralJournalPdf(m, writeToZip)
	ChartOfAccountsPdf(m, writeToZip)
	addBillImagesToZip(m, writeToZip)
	err = zipWriter.Close()
	if err != nil {
		log.Fatal(err)
	}
	//io.Copy(outerWriter, zipBuf)

}
