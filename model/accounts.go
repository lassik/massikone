package model

import (
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
)

const accountNestingLevel = 9

type Account struct {
	RawAccountID string
	AccountID    string
	Prefix       string
	Title        string
	HTagLevel    string
	IsMatch      bool
}

func (m *Model) GetAccounts(usedOnly bool, matchAccountID string) []Account {
	noAccounts := []Account{}
	accounts := noAccounts
	//populateAccounts()
	rows, err := sq.Select("account_id, title, nesting_level").
		From("period_account").OrderBy("account_id, nesting_level").
		RunWith(m.tx).Query()
	if m.isErr(err) {
		return noAccounts
	}
	defer rows.Close()
	for rows.Next() {
		var a Account
		var nestingLevel int
		if m.isErr(rows.Scan(&a.RawAccountID, &a.Title, &nestingLevel)) {
			return noAccounts
		}
		isAccount := (nestingLevel == accountNestingLevel)
		dashLevel := 0
		a.HTagLevel = ""
		if !isAccount {
			dashLevel = 1 + nestingLevel
			a.HTagLevel = strconv.Itoa(2 + nestingLevel)
		}
		a.Prefix = a.RawAccountID
		if !isAccount {
			a.Prefix = strings.Repeat("=", dashLevel)
		}
		a.AccountID = ""
		if isAccount {
			a.AccountID = a.RawAccountID
		}
		a.IsMatch = (matchAccountID != "" &&
			a.AccountID == matchAccountID)
		accounts = append(accounts, a)
	}
	if usedOnly {
		//accounts = reject_unused_accounts(accounts, @db[:bill_entry].select(:account_id).order(:account_id).distinct.map(:account_id))
	}
	return accounts
}

func (m *Model) GetAccountLookup() map[int]string {
	lookup := map[int]string{}
	rows, err := sq.Select("account_id, title").
		From("period_account").OrderBy("account_id").
		Where(sq.Eq{"nesting_level": accountNestingLevel}).
		RunWith(m.tx).Query()
	if m.isErr(err) {
		return lookup
	}
	defer rows.Close()
	for rows.Next() {
		var accountID int
		var title string
		rows.Scan(&accountID, &title)
		lookup[accountID] = title
	}
	return lookup
}
