package reports

import ()

func BalanceSheetPdf(getWriter GetWriter) {
	blankPdf(getWriter, "tase")
}

func BalanceSheetDetailedPdf(getWriter GetWriter) {
	blankPdf(getWriter, "tase erittelyin")
}
