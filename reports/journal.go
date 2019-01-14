package reports

import (
	"fmt"

	"github.com/lassik/massikone/model"
)

func GeneralJournalPdf(m *model.Model, getWriter GetWriter) {
	const numberWidth = 2
	const accountWidth = 8
	const descriptionWidth = 10
	acctMap := m.GetAccountMap()
	doc := document{
		title:     "Päiväkirja",
		filename:  "päiväkirja",
		orgName:   m.GetSettings().OrgShortName,
		period:    "1.1.2018 - 31.12.2018",
		printDate: "1.12.2018",
		headerRow: []cell{
			cell{text: "Nro", width: numberWidth},
			cell{text: "Pvm/Tili", width: accountWidth},
			cell{text: "Debet", width: numberWidth, rightAlign: true},
			cell{text: "Kredit", width: numberWidth, rightAlign: true},
			cell{text: "Selite", width: descriptionWidth},
		},
	}
	journal := m.GetJournal()
	for _, document := range journal.Documents {
		doc.rows = append(doc.rows, []cell{
			cell{
				text:  document.DocumentID,
				width: numberWidth,
			},
			cell{
				text: document.PaidDateFi,
				width: accountWidth + 2*numberWidth +
					descriptionWidth,
			},
		})
		for _, entry := range document.Entries {
			debit := ""
			credit := ""
			if entry.IsDebit {
				debit = entry.Amount
			} else {
				credit = entry.Amount
			}
			doc.rows = append(doc.rows, []cell{
				cell{
					text:  "",
					width: numberWidth,
				},
				cell{
					text: fmt.Sprintf("%d %s",
						entry.AccountID,
						acctMap[entry.AccountID].Title,
					),
					width:       accountWidth,
					indentLevel: 1,
				},
				cell{
					text:       debit,
					width:      numberWidth,
					rightAlign: true,
				},
				cell{
					text:       credit,
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
	doc.rows = append(doc.rows, []cell{
		cell{width: numberWidth},
		cell{width: accountWidth},
		cell{
			text:       amountFromCents(journal.TotalDebitCents),
			width:      numberWidth,
			rightAlign: true,
			bold:       true,
		},
		cell{
			text:       amountFromCents(journal.TotalCreditCents),
			width:      numberWidth,
			rightAlign: true,
			bold:       true,
		},
		cell{width: descriptionWidth},
	})
	writePdf(m, doc, getWriter)
}
