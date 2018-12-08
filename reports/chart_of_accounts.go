package reports

import (
	"github.com/lassik/massikone/model"
)

func ChartOfAccountsPdf(m *model.Model, getWriter GetWriter) {
	accounts := m.GetAccountList(false, "")
	doc := document{
		orgName:   m.GetSettings().OrgShortName,
		title:     "Tilikartta",
		filename:  "tilikartta",
		period:    "1.1.2018 - 31.12.2018",
		printDate: "1.12.2018",
	}
	for _, acct := range accounts {
		bold := acct.IsHeading()
		thisRow := []cell{
			cell{
				text:  acct.AccountIDStr,
				bold:  bold,
				width: 1,
			},
			cell{
				text:        acct.Title,
				bold:        bold,
				indentLevel: acct.NestingLevel,
				width:       10,
			},
		}
		doc.rows = append(doc.rows, thisRow)
	}
	writePdf(m, doc, getWriter)
}
