package reports

import (
	"../model"
)

func GeneralJournalPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(getWriter, "päiväkirja")
}
