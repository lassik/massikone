package reports

import (
	"github.com/lassik/massikone/model"
)

func GeneralJournalPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(getWriter, "päiväkirja")
}
