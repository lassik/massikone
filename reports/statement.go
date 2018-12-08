package reports

import (
	"github.com/lassik/massikone/model"
)

func IncomeStatementPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(m, getWriter, "tuloslaskelma")
}

func IncomeStatementDetailedPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(m, getWriter, "tuloslaskelma erittelyin")
}

func BalanceSheetPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(m, getWriter, "tase")
}

func BalanceSheetDetailedPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(m, getWriter, "tase erittelyin")
}
