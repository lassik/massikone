package model

import ()

func entrySubtractsFromBalance(entryIsDebit bool, acctType int) bool {
	if acctType == AssetAccount || acctType == ExpenseAccount {
		return !entryIsDebit
	}
	return entryIsDebit
}

func GetAccountBalancesAndProfit(acctMap map[int]Account,
	entries []DocumentEntry) (map[int]int64, int64) {
	balances := map[int]int64{}
	var profit int64
	for _, entry := range entries {
		acctID := entry.AccountID
		acctType := acctMap[acctID].AccountType
		cents := entry.UnitCount * entry.UnitCostCents
		if entrySubtractsFromBalance(entry.IsDebit, acctType) {
			balances[acctID] -= cents
		} else {
			balances[acctID] += cents
		}
		if acctType == ExpenseAccount {
			profit -= cents
		} else if acctType == RevenueAccount {
			profit += cents
		}
	}
	return balances, profit
}

func GetAccountRangeBalance(acctMap map[int]Account,
	balances map[int]int64, ranges []AccountRange) int64 {
	var rangesBalance int64
	for acctID, balance := range balances {
		if AccountIDInRange(acctID, ranges) {
			if acctMap[acctID].AccountType == ExpenseAccount {
				rangesBalance -= balance
			} else {
				rangesBalance += balance
			}
		}
	}
	return rangesBalance
}
