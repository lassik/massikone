package reports

import (
	"../model"
)

func ChartOfAccountsPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(getWriter, "tilikartta")
}
