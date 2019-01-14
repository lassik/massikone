package model

import ()

type Journal struct {
	Documents        []Document
	TotalDebitCents  int64
	TotalCreditCents int64
}

func (m *Model) GetJournal() Journal {
	journal := Journal{}
	if !m.isAdmin() {
		return journal
	}
	q := selectDocument()
	rows, err := q.RunWith(m.tx).Query()
	if m.isErr(err) {
		return journal
	}
	defer rows.Close()
	documents := []Document{}
	var totalDebitCents int64
	var totalCreditCents int64
	for rows.Next() {
		document, err := scanDocument(rows)
		if m.isErr(err) {
			return journal
		}
		m.populateDocumentEntries(&document)
		for _, documentEntry := range document.Entries {
			cents := documentEntry.UnitCount * documentEntry.UnitCostCents
			if documentEntry.IsDebit {
				totalDebitCents += cents
			} else {
				totalCreditCents += cents
			}
		}
		documents = append(documents, document)
	}
	if m.isErr(rows.Err()) {
		return journal
	}
	journal.Documents = documents
	journal.TotalDebitCents = totalDebitCents
	journal.TotalCreditCents = totalCreditCents
	return journal
}
