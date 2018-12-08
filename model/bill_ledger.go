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

type Ledger struct {
	Accounts         []LedgerAccount
	TotalDebitCents  int64
	TotalCreditCents int64
}

func (m *Model) GetLedger() Ledger {
	ledger := Ledger{}
	if !m.isAdmin() {
		return ledger
	}
	acctMap := m.GetAccountMap()
	rows, err := selectBill().RunWith(m.tx).Query()
	if m.isErr(err) {
		return ledger
	}
	defer rows.Close()
	ledgerMap := map[int]LedgerAccount{}
	var totalDebitCents int64
	var totalCreditCents int64
	for rows.Next() {
		bill, err := scanBill(rows)
		if err != nil {
			return ledger
		}
		m.populateBillEntries(&bill)
		for _, billEntry := range bill.Entries {
			cents := billEntry.UnitCount * billEntry.UnitCostCents
			ledgerAccount := ledgerMap[billEntry.AccountID]
			ledgerAccount.AccountID = billEntry.AccountID
			ledgerAccount.AccountTitle =
				acctMap[ledgerAccount.AccountID].Title
			ledgerAccount.CurrentBalanceCents += cents
			ledgerEntry := LedgerEntry{}
			if billEntry.IsDebit {
				ledgerEntry.DebitAmount = billEntry.Amount
				totalDebitCents += cents
			} else {
				ledgerEntry.CreditAmount = billEntry.Amount
				totalCreditCents += cents
			}
			ledgerEntry.BalanceAfter = amountFromCents(
				ledgerAccount.CurrentBalanceCents)
			ledgerEntry.BillID = bill.BillID
			ledgerEntry.PaidDateISO = bill.PaidDateISO
			ledgerEntry.PaidDateFi = bill.PaidDateFi
			ledgerEntry.Description = billEntry.Description
			ledgerAccount.Entries =
				append(ledgerAccount.Entries, ledgerEntry)
			ledgerMap[billEntry.AccountID] = ledgerAccount
		}
	}
	if m.isErr(rows.Err()) {
		return ledger
	}
	acctList := []LedgerAccount{}
	for _, ledgerAccount := range ledgerMap {
		acctList = append(acctList, ledgerAccount)
	}
	sort.Slice(acctList, func(i, j int) bool {
		return acctList[i].AccountID < acctList[j].AccountID
	})
	ledger.Accounts = acctList
	ledger.TotalDebitCents = totalDebitCents
	ledger.TotalCreditCents = totalCreditCents
	return ledger
}
