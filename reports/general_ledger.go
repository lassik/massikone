package reports

import (
	"../model"
)

func GeneralLedgerPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(getWriter, "pääkirja")
}
