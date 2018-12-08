package model

import ()

func (m *Model) GetBillsForJournal() []Bill {
	noBills := []Bill{}
	bills := noBills
	if !m.isAdmin() {
		return noBills
	}
	q := selectBill()
	rows, err := q.RunWith(m.tx).Query()
	if m.isErr(err) {
		return noBills
	}
	defer rows.Close()
	for rows.Next() {
		bill, err := scanBill(rows)
		if m.isErr(err) {
			return noBills
		}
		m.populateBillEntries(&bill)
		bills = append(bills, bill)
	}
	if m.isErr(rows.Err()) {
		return noBills
	}
	return bills
}
