package model

import (
	sq "github.com/Masterminds/squirrel"
)

type Settings struct {
	OrgFullName  string
	OrgShortName string
}

func getSettings(runner sq.BaseRunner) (Settings, error) {
	settings := Settings{}
	rows, err := sq.Select("name, value").From("setting").
		RunWith(runner).Query()
	if err != nil {
		return settings, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var value string
		if err = rows.Scan(&name, &value); err != nil {
			return settings, err
		}
		switch name {
		case "org_full_name":
			settings.OrgFullName = value
		case "org_short_name":
			settings.OrgShortName = value
		}
	}
	return settings, rows.Err()
}

func (m *Model) GetSettings() Settings {
	settings, err := getSettings(m.tx)
	m.isErr(err)
	return settings
}

func GetSettings() Settings {
	settings, _ := getSettings(getDB())
	return settings
}
