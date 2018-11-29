package model

import (
	sq "github.com/Masterminds/squirrel"
)

type Settings struct {
	OrgFullName  string
	OrgShortName string
}

func getSetting(settings *Settings, name, value string) {
	switch name {
	case "OrgFullName":
		settings.OrgFullName = value
	case "OrgShortName":
		settings.OrgShortName = value
	}
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
		getSetting(&settings, name, value)
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
