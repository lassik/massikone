package reports

import (
	"../model"
)

func IncomeStatementPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(getWriter, "tuloslaskelma")
}

func IncomeStatementDetailedPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(getWriter, "tuloslaskelma erittelyin")
}
