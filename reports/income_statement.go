package reports

import ()

func IncomeStatementPdf(getWriter GetWriter) {
	blankPdf(getWriter, "tuloslaskelma")
}

func IncomeStatementDetailedPdf(getWriter GetWriter) {
	blankPdf(getWriter, "tuloslaskelma erittelyin")
}
