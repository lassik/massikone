package model

import (
	sq "github.com/Masterminds/squirrel"
	"strconv"
	"strings"
)

const AccountNestingLevel = 9

func (m *Model) GetAccounts(usedOnly bool) []map[string]interface{} {
	var accounts []map[string]interface{}
	//populateAccounts()
	rows, err := sq.Select("account_id, title, nesting_level").
		From("period_account").OrderBy("account_id, nesting_level").
		RunWith(m.tx).Query()
	if m.isErr(err) {
                return accounts
        }
	defer rows.Close()
	for rows.Next() {
		var account_id string
		var title string
		var nesting_level int
		if m.isErr(rows.Scan(&account_id, &title, &nesting_level)) {
                        return accounts
                }
		is_account := (nesting_level == AccountNestingLevel)
		dash_level, htag_level := 0, ""
		if !is_account {
			dash_level = 1 + nesting_level
			htag_level = strconv.Itoa(2 + nesting_level)
		}
		prefix := account_id
		if !is_account {
			prefix = strings.Repeat("=", dash_level)
		}
		account_id_or_nil := ""
		if is_account {
			account_id_or_nil = account_id
		}
		accounts = append(accounts,
			map[string]interface{}{
				"raw_account_id": account_id,
				"account_id":     account_id_or_nil,
				"title":          title,
				"prefix":         prefix,
				"htag_level":     htag_level,
			})
	}
	if usedOnly {
		//accounts = reject_unused_accounts(accounts, @db[:bill_entry].select(:account_id).order(:account_id).distinct.map(:account_id))
	}
	return accounts
}
