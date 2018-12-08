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
	AccountIDStr string
	Prefix       string
	Title        string
	HTagLevel    string
	IsMatch      bool
}

func (m *Model) GetAccounts(usedOnly bool, matchAccountID string) []Account {
	noAccounts := []Account{}
	accounts := noAccounts
	//populateAccounts()
	rows, err := sq.Select("account_id, account_type, title, nesting_level").
		From("period_account").OrderBy("account_id, nesting_level").
		RunWith(m.tx).Query()
	if m.isErr(err) {
		return noAccounts
	}
	defer rows.Close()
	for rows.Next() {
		var a Account
		var nestingLevel int
		if m.isErr(rows.Scan(&a.AccountID, &a.AccountType,
			&a.Title, &nestingLevel)) {
			return noAccounts
		}
		isAccount := (nestingLevel == accountNestingLevel)
		dashLevel := 0
		if !isAccount {
			dashLevel = 1 + nestingLevel
			a.HTagLevel = strconv.Itoa(2 + nestingLevel)
		}
		if isAccount {
			a.AccountIDStr = strconv.Itoa(a.AccountID)
			a.Prefix = a.AccountIDStr
		} else {
			a.Prefix = strings.Repeat("=", dashLevel)
		}
		a.IsMatch = (matchAccountID != "" &&
			a.AccountIDStr == matchAccountID)
		accounts = append(accounts, a)
	}
	if usedOnly {
		//accounts = reject_unused_accounts(accounts, @db[:bill_entry].select(:account_id).order(:account_id).distinct.map(:account_id))
	}
	return accounts
}

func (m *Model) GetAccountMap() map[int]Account {
	acctMap := map[int]Account{}
	rows, err := sq.Select("account_id, title").
		From("period_account").OrderBy("account_id").
		Where(sq.Eq{"nesting_level": accountNestingLevel}).
		RunWith(m.tx).Query()
	if m.isErr(err) {
		return acctMap
	}
	defer rows.Close()
	for rows.Next() {
		var acctID int
		acct := Account{}
		rows.Scan(&acctID, &acct.Title)
		acctMap[acctID] = acct
	}
	return acctMap
}
