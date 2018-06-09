package model

import (
	sq "github.com/Masterminds/squirrel"
)

type Preferences struct {
	OrgFullName  string
	OrgShortName string
}

func getPreferences(runner sq.BaseRunner) (Preferences, error) {
	prefs := Preferences{}
	rows, err := sq.Select("name, value").From("preference").
		RunWith(runner).Query()
	if err != nil {
		return prefs, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var value string
		if err = rows.Scan(&name, &value); err != nil {
			return prefs, err
		}
		switch name {
		case "org_full_name":
			prefs.OrgFullName = value
		case "org_short_name":
			prefs.OrgShortName = value
		}
	}
	return prefs, rows.Err()
}

func (m *Model) GetPreferences() Preferences {
	prefs, err := getPreferences(m.tx)
	m.isErr(err)
	return prefs
}

func GetPreferences() Preferences {
	prefs, _ := getPreferences(getDB())
	return prefs
}
