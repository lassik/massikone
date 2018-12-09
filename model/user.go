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

func countUsers(tx *sql.Tx) int {
	var count int
	sq.Select("count(*)").From("user").
		RunWith(tx).Limit(1).QueryRow().Scan(&count)
	return count
}

func getUserIDByAuth(tx *sql.Tx, authProvider, authUserID string) int64 {
	var userID int64
	sq.Select("user_id").From("user_auth").Where(sq.Eq{
		"auth_provider": authProvider,
		"auth_user_id":  authUserID,
	}).RunWith(tx).Limit(1).QueryRow().Scan(&userID)
	return userID
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

func selectUser() sq.SelectBuilder {
	return sq.Select("user_id, full_name, is_admin").
		From("user").
		OrderBy("lower(full_name)")
}

func scanUser(rows sq.RowScanner) (User, error) {
	var user User
	err := rows.Scan(&user.UserID, &user.FullName, &user.IsAdmin)
	return user, err
}

func (m *Model) getUserByID(userID int64) (User, error) {
	return scanUser(selectUser().Where(sq.Eq{"user_id": userID}).
		RunWith(m.tx).QueryRow())
}

func (m *Model) GetUsers(matchUserID int64) []User {
	var noUsers []User
	if !m.isAdmin() {
		return noUsers
	}
	rows, err := selectUser().RunWith(m.tx).Query()
	if m.isErr(err) {
		return noUsers
	}
	defer rows.Close()
	users := noUsers
	for rows.Next() {
		user, err := scanUser(rows)
		if m.isErr(err) {
			return noUsers
		}
		user.IsMatch = (matchUserID != 0 && user.UserID == matchUserID)
		users = append(users, user)
	}
	sort.SliceStable(users, func(i, j int) bool {
		return strings.ToLower(users[i].FullName) <
			strings.ToLower(users[j].FullName)
	})
	return users
}

func GetOrPutUser(authProvider, authUserID, fullName string) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	userID := getUserIDByAuth(tx, authProvider, authUserID)
	if userID == 0 {
		err = sq.Select("coalesce(max(user_id), 0) + 1").From("user").
			RunWith(tx).Limit(1).QueryRow().Scan(&userID)
		if err != nil {
			return 0, err
		}
		_, err = sq.Insert("user").SetMap(sq.Eq{
			"user_id":   userID,
			"full_name": fullName,
			"is_admin":  (countUsers(tx) == 0),
		}).RunWith(tx).Exec()
		if err != nil {
			return 0, err
		}
		_, err = sq.Insert("user_auth").SetMap(sq.Eq{
			"user_id":       userID,
			"auth_provider": authProvider,
			"auth_user_id":  authUserID,
		}).RunWith(tx).Exec()
	} else {
		_, err = sq.Update("user").SetMap(sq.Eq{
			"full_name": fullName,
		}).Where(sq.Eq{"user_id": userID}).RunWith(tx).Exec()
	}
	if err != nil {
		return 0, err
	}
	err = tx.Commit()
	if err != nil {
		return 0, err
	}
	return userID, err
}
