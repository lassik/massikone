package model

import (
	"sort"
)

type LedgerEntry struct {
	DocumentID   string
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
	rows, err := selectDocument().RunWith(m.tx).Query()
	if m.isErr(err) {
		return ledger
	}
	defer rows.Close()
	ledgerMap := map[int]LedgerAccount{}
	var totalDebitCents int64
	var totalCreditCents int64
	for rows.Next() {
		document, err := scanDocument(rows)
		if err != nil {
			return ledger
		}
		m.populateDocumentEntries(&document)
		for _, documentEntry := range document.Entries {
			cents := documentEntry.UnitCount * documentEntry.UnitCostCents
			ledgerAccount := ledgerMap[documentEntry.AccountID]
			ledgerAccount.AccountID = documentEntry.AccountID
			ledgerAccount.AccountTitle =
				acctMap[ledgerAccount.AccountID].Title
			ledgerEntry := LedgerEntry{}
			if documentEntry.IsDebit {
				ledgerEntry.DebitAmount = documentEntry.Amount
				ledgerAccount.CurrentBalanceCents += cents
				totalDebitCents += cents
			} else {
				ledgerEntry.CreditAmount = documentEntry.Amount
				ledgerAccount.CurrentBalanceCents -= cents
				totalCreditCents += cents
			}
			ledgerEntry.BalanceAfter = amountFromCents(
				ledgerAccount.CurrentBalanceCents)
			ledgerEntry.DocumentID = document.DocumentID
			ledgerEntry.PaidDateISO = document.PaidDateISO
			ledgerEntry.PaidDateFi = document.PaidDateFi
			ledgerEntry.Description = documentEntry.Description
			ledgerAccount.Entries =
				append(ledgerAccount.Entries, ledgerEntry)
			ledgerMap[documentEntry.AccountID] = ledgerAccount
		}
	}
	if m.isErr(rows.Err()) {
		return ledger
	}
	acctList := []LedgerAccount{}
	for _, ledgerAccount := range ledgerMap {
		ents := ledgerAccount.Entries
		sort.Slice(ents, func(i, j int) bool {
			if ents[i].PaidDateISO < ents[j].PaidDateISO {
				return true
			}
			if ents[i].PaidDateISO > ents[j].PaidDateISO {
				return false
			}
			return ents[i].DocumentID < ents[j].DocumentID
		})
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
