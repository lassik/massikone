package reports

import (
	"github.com/lassik/massikone/model"
)

func GeneralLedgerPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(getWriter, "pääkirja")
}
