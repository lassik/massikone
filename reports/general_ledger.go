package reports

import (
	"strconv"

	"github.com/lassik/massikone/model"
)

func GeneralLedgerPdf(m *model.Model, getWriter GetWriter) {
	const dateWidth = 3
	const numberWidth = 2
	const accountWidth = 8
	const descriptionWidth = 10
	doc := document{
		title:     "Pääkirja",
		filename:  "pääkirja",
		orgName:   m.GetSettings().OrgShortName,
		period:    "1.1.2018 - 31.12.2018",
		printDate: "1.12.2018",
		headerRow: []cell{
			cell{text: "Tili", width: numberWidth},
			cell{text: "Tili/Tosite", width: dateWidth},
			cell{text: "Pvm", width: dateWidth},
			cell{text: "Debet", width: numberWidth, rightAlign: true},
			cell{text: "Kredit", width: numberWidth, rightAlign: true},
			cell{text: "Saldo", width: numberWidth, rightAlign: true},
			cell{text: "Selite", width: descriptionWidth},
		},
	}
	for _, account := range m.GetLedger() {
		doc.rows = append(doc.rows, []cell{
			cell{
				text:  strconv.Itoa(account.AccountID),
				width: numberWidth,
			},
			cell{
				text: account.AccountTitle,
				width: accountWidth + 2*numberWidth +
					descriptionWidth,
			},
		})
		if account.StartingBalance != "" {
			doc.rows = append(doc.rows, []cell{
				cell{width: numberWidth},
				cell{width: numberWidth},
				cell{width: dateWidth},
				cell{width: numberWidth},
				cell{width: numberWidth},
				cell{
					text:       account.StartingBalance,
					width:      numberWidth,
					rightAlign: true,
				},
				cell{
					text:  "Alkusaldo",
					width: descriptionWidth,
				},
			})
		}
		for _, entry := range account.Entries {
			doc.rows = append(doc.rows, []cell{
				cell{
					text:  "",
					width: numberWidth,
				},
				cell{
					text:  entry.BillID,
					width: numberWidth,
				},
				cell{
					text:       entry.PaidDateFi,
					width:      dateWidth,
					rightAlign: true,
				},
				cell{
					text:       entry.DebitAmount,
					width:      numberWidth,
					rightAlign: true,
				},
				cell{
					text:       entry.CreditAmount,
					width:      numberWidth,
					rightAlign: true,
				},
				cell{
					text:       entry.BalanceAfter,
					width:      numberWidth,
					rightAlign: true,
				},
				cell{
					text:  entry.Description,
					width: descriptionWidth,
				},
			})
		}
	}
	writePdf(doc, getWriter)
}
