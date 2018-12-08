package model

import ()

type Journal struct {
	Bills            []Bill
	TotalDebitCents  int64
	TotalCreditCents int64
}

func (m *Model) GetJournal() Journal {
	journal := Journal{}
	if !m.isAdmin() {
		return journal
	}
	q := selectBill()
	rows, err := q.RunWith(m.tx).Query()
	if m.isErr(err) {
		return journal
	}
	defer rows.Close()
	bills := []Bill{}
	var totalDebitCents int64
	var totalCreditCents int64
	for rows.Next() {
		bill, err := scanBill(rows)
		if m.isErr(err) {
			return journal
		}
		m.populateBillEntries(&bill)
		for _, billEntry := range bill.Entries {
			cents := billEntry.UnitCount * billEntry.UnitCostCents
			if billEntry.IsDebit {
				totalDebitCents += cents
			} else {
				totalCreditCents += cents
			}
		}
		bills = append(bills, bill)
	}
	if m.isErr(rows.Err()) {
		return journal
	}
	journal.Bills = bills
	journal.TotalDebitCents = totalDebitCents
	journal.TotalCreditCents = totalCreditCents
	return journal
}
