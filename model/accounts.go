package model

import (
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
)

const accountNestingLevel = 9

const (
	AssetAccount      = 0 // Vastaavaa
	LiabilityAccount  = 1 // Vastattavaa
	EquityAccount     = 2 // Oma pääoma
	RevenueAccount    = 3 // Tulot
	ExpenseAccount    = 4 // Menot
	PastProfitAccount = 5 // Edellisten tilikausien voitto
	ProfitAccount     = 6 // Tilikauden voitto
)

type Account struct {
	AccountID    int
	AccountType  int
	NestingLevel int
	AccountIDStr string
	Prefix       string
	Title        string
	IsMatch      bool
}

func (acct Account) IsHeading() bool {
	return acct.NestingLevel != accountNestingLevel
}

func selectAccount() sq.SelectBuilder {
	return sq.Select("account_id, account_type, title, nesting_level").
		From("period_account").
		OrderBy("account_id, nesting_level")
}

func scanAccount(rows sq.RowScanner) (Account, error) {
	var a Account
	if err := rows.Scan(&a.AccountID, &a.AccountType,
		&a.Title, &a.NestingLevel); err != nil {
		return a, err
	}
	if a.IsHeading() {
		a.Prefix = strings.Repeat("=", a.NestingLevel+1)
	} else {
		a.AccountIDStr = strconv.Itoa(a.AccountID)
		a.Prefix = a.AccountIDStr
	}
	return a, nil
}

func (m *Model) GetAccountList(usedOnly bool, matchAccountID string) []Account {
	noAccounts := []Account{}
	accounts := noAccounts
	rows, err := selectAccount().RunWith(m.tx).Query()
	if m.isErr(err) {
		return noAccounts
	}
	defer rows.Close()
	for rows.Next() {
		acct, err := scanAccount(rows)
		if m.isErr(err) {
			return noAccounts
		}
		acct.IsMatch = (matchAccountID != "" &&
			acct.AccountIDStr == matchAccountID)
		accounts = append(accounts, acct)
	}
	if usedOnly {
		//accounts = reject_unused_accounts(accounts, @db[:bill_entry].select(:account_id).order(:account_id).distinct.map(:account_id))
	}
	return accounts
}

func (m *Model) GetAccountMap() map[int]Account {
	acctMap := map[int]Account{}
	rows, err := selectAccount().RunWith(m.tx).Query()
	if m.isErr(err) {
		return acctMap
	}
	defer rows.Close()
	for rows.Next() {
		acct, err := scanAccount(rows)
		if m.isErr(err) {
			break
		}
		if acct.IsHeading() {
			continue
		}
		acctID, _ := strconv.Atoi(acct.AccountIDStr)
		acctMap[acctID] = acct
	}
	return acctMap
}
