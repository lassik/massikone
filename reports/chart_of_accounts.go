package reports

import (
	"github.com/lassik/massikone/model"
)

func ChartOfAccountsPdf(m *model.Model, getWriter GetWriter) {
	blankPdf(getWriter, "tilikartta")
}
