package model

import (
	"database/sql"
	"errors"
	"sort"
	"strings"

	sq "github.com/Masterminds/squirrel"
)

type User struct {
	UserID   int64
	FullName string
	IsAdmin  bool
	IsMatch  bool
}

func (m *Model) Forbidden() {
	m.isErr(errors.New("Forbidden"))
}

func (m *Model) isAdmin() bool {
	if m.user.IsAdmin {
		return true
	}
	m.Forbidden()
	return false
}

func (m *Model) isAdminOrUser(userID int64) bool {
	if m.user.IsAdmin || ((userID != 0) && (m.user.UserID == userID)) {
		return true
	}
	m.Forbidden()
	return false
}

func (m *Model) getUserByID(userID int64) (*User, error) {
	var user User
	q := sq.Select("user_id, full_name, is_admin").
		From("user").Where(sq.Eq{"user_id": userID})
	err := q.RunWith(m.tx).Limit(1).QueryRow().Scan(
		&user.UserID, &user.FullName, &user.IsAdmin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (m *Model) GetUsers(matchUserID int64) []User {
	var noUsers []User
	if !m.isAdmin() {
		return noUsers
	}
	rows, err := sq.Select("user_id, full_name, is_admin").
		From("user").OrderBy("user_id").RunWith(m.tx).Query()
	if m.isErr(err) {
		return noUsers
	}
	defer rows.Close()
	users := noUsers
	for rows.Next() {
		var u User
		if m.isErr(rows.Scan(&u.UserID, &u.FullName, &u.IsAdmin)) {
			return noUsers
		}
		u.IsMatch = (matchUserID != 0 && u.UserID == matchUserID)
		users = append(users, u)
	}
	sort.SliceStable(users, func(i, j int) bool {
		return strings.ToLower(users[i].FullName) <
			strings.ToLower(users[j].FullName)
	})
	return users
}

func GetOrPutUser(provider, providerUserID, email, fullName string) (string, error) {
	providerUserIDColumn := "user_id_" + provider
	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	setmap := sq.Eq{
		providerUserIDColumn: providerUserID,
		"email":              email,
		"full_name":          fullName,
	}
	result, err := sq.Update("user").SetMap(setmap).
		Where(sq.Eq{providerUserIDColumn: providerUserID}).
		RunWith(tx).Exec()
	if err != nil {
		return "", err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	if count < 1 {
		var oldUserCount int
		err = sq.Select("count(*)").From("user").
			RunWith(tx).Limit(1).QueryRow().Scan(&oldUserCount)
		if err != nil {
			return "", err
		}
		isAdmin := (oldUserCount == 0)
		setmap["is_admin"] = isAdmin
		sq.Insert("user").SetMap(setmap).RunWith(tx).QueryRow().Scan()
	}
	var userID string
	err = sq.Select("user_id").From("user").
		Where(sq.Eq{providerUserIDColumn: providerUserID}).
		RunWith(tx).Limit(1).QueryRow().Scan(&userID)
	if err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	return userID, err
}
