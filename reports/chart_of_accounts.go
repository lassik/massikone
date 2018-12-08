package reports

import (
	"strconv"

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
		level := 0
		if acct.HTagLevel != "" {
			level, _ = strconv.Atoi(acct.HTagLevel)
		}
		if level < 1 {
			level = 9
		}
		bold := level < 9
		thisRow := []cell{
			cell{
				text:  acct.AccountIDStr,
				bold:  bold,
				width: 1,
			},
			cell{
				text:        acct.Title,
				bold:        bold,
				indentLevel: level,
				width:       10,
			},
		}
		doc.rows = append(doc.rows, thisRow)
	}
	writePdf(doc, getWriter)
}
