package reports

import (
	"github.com/lassik/massikone/model"
)

func BalanceSheetPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(getWriter, "tase")
}

func BalanceSheetDetailedPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(getWriter, "tase erittelyin")
}
