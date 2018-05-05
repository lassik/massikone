package reports

import (
        "../model"
)

func BalanceSheetPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(getWriter, "tase")
}

func BalanceSheetDetailedPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(getWriter, "tase erittelyin")
}
