package model

import (
	"sort"
)

type LedgerEntry struct {
	BillID       string
	PaidDateISO  string
	PaidDateFi   string
	DebitAmount  string
	CreditAmount string
	BalanceAfter string
	Description  string
}

type LedgerAccount struct {
	AccountID           int
	AccountTitle        string
	StartingBalance     string
	CurrentBalanceCents int64
	Entries             []LedgerEntry
}

func (m *Model) GetLedger() []LedgerAccount {
	emptyLedger := []LedgerAccount{}
	if !m.isAdmin() {
		return emptyLedger
	}
	acctMap := m.GetAccountMap()
	rows, err := selectBill().RunWith(m.tx).Query()
	if m.isErr(err) {
		return emptyLedger
	}
	defer rows.Close()
	ledgerMap := map[int]LedgerAccount{}
	for rows.Next() {
		bill, err := scanBill(rows)
		if err != nil {
			return emptyLedger
		}
		m.populateBillEntries(&bill)
		for _, billEntry := range bill.Entries {
			ledgerAccount := ledgerMap[billEntry.AccountID]
			ledgerAccount.AccountID = billEntry.AccountID
			ledgerAccount.AccountTitle =
				acctMap[ledgerAccount.AccountID].Title
			ledgerAccount.CurrentBalanceCents +=
				billEntry.UnitCount * billEntry.UnitCostCents
			ledgerEntry := LedgerEntry{}
			ledgerEntry.BalanceAfter = amountFromCents(
				ledgerAccount.CurrentBalanceCents)
			ledgerEntry.BillID = bill.BillID
			ledgerEntry.PaidDateISO = bill.PaidDateISO
			ledgerEntry.PaidDateFi = bill.PaidDateFi
			if billEntry.IsDebit {
				ledgerEntry.DebitAmount = billEntry.Amount
			} else {
				ledgerEntry.CreditAmount = billEntry.Amount
			}
			ledgerEntry.Description = billEntry.Description
			ledgerAccount.Entries =
				append(ledgerAccount.Entries, ledgerEntry)
			ledgerMap[billEntry.AccountID] = ledgerAccount
		}
	}
	if m.isErr(rows.Err()) {
		return emptyLedger
	}
	ledger := []LedgerAccount{}
	for _, ledgerAccount := range ledgerMap {
		ledger = append(ledger, ledgerAccount)
	}
	sort.Slice(ledger, func(i, j int) bool {
		return ledger[i].AccountID < ledger[j].AccountID
	})
	return ledger
}
