package model

import (
	"crypto/sha1"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	sq "github.com/Masterminds/squirrel"
)

const (
	NoPermission      = 0
	NormalPermission  = 1
	ViewAllPermission = 2
	AdminPermission   = 3
)

type User struct {
	UserID          int64
	FullName        string
	PermissionLevel int
	IsAdmin         bool
	IsMatch         bool
}

func getPrivateSessionUser() User {
	return User{UserID: 0, PermissionLevel: AdminPermission, IsAdmin: true}
}

func countUsers(tx *sql.Tx) (count int) {
	sq.Select("count(*)").From("user").
		RunWith(tx).Limit(1).QueryRow().Scan(&count)
	return
}

func getNewUserID(tx *sql.Tx) (userID int64, err error) {
	err = sq.Select("coalesce(max(user_id), 0) + 1").From("user").
		RunWith(tx).Limit(1).QueryRow().Scan(&userID)
	return
}

func getUserIDByAuth(tx *sql.Tx, authProvider, authUserID string) (userID int64) {
	sq.Select("user_id").From("user_auth").Where(sq.Eq{
		"auth_provider": authProvider,
		"auth_user_id":  authUserID,
	}).RunWith(tx).Limit(1).QueryRow().Scan(&userID)
	return
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
	return sq.Select("user_id, full_name, permission_level").
		From("user").
		OrderBy("lower(full_name)")
}

func scanUser(rows sq.RowScanner) (User, error) {
	var user User
	err := rows.Scan(&user.UserID, &user.FullName, &user.PermissionLevel)
	user.IsAdmin = (user.PermissionLevel >= AdminPermission)
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

func insertUser(tx *sql.Tx, fullName string) (userID int64, err error) {
	permissionLevel := NormalPermission
	if countUsers(tx) == 0 {
		permissionLevel = AdminPermission
	}
	userID, err = getNewUserID(tx)
	if err != nil {
		return 0, err
	}
	if _, err := sq.Insert("user").SetMap(sq.Eq{
		"user_id":          userID,
		"full_name":        fullName,
		"permission_level": permissionLevel,
	}).RunWith(tx).Exec(); err != nil {
		return 0, err
	}
	return userID, nil
}

func insertUserAuth(tx *sql.Tx, userID int64,
	authProvider, authUserID string) (err error) {
	_, err = sq.Insert("user_auth").SetMap(sq.Eq{
		"user_id":       userID,
		"auth_provider": authProvider,
		"auth_hash":     "sha1",
		"auth_user_id":  authUserID,
	}).RunWith(tx).Exec()
	return
}

func updateUserFullName(tx *sql.Tx, userID int64, fullName string) (err error) {
	_, err = sq.Update("user").SetMap(sq.Eq{
		"full_name": fullName,
	}).Where(sq.Eq{"user_id": userID}).RunWith(tx).Exec()
	return
}

func hashAuthUserID(authProvider, authUserID string) string {
	hasher := sha1.New()
	hasher.Write([]byte(authProvider))
	hasher.Write([]byte(authUserID))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func GetOrPutUser(authProvider, authUserID,
	fullName string) (userID int64, err error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	authUserID = hashAuthUserID(authProvider, authUserID)
	userID = getUserIDByAuth(tx, authProvider, authUserID)
	if userID == 0 {
		if userID, err = insertUser(tx, fullName); err != nil {
			return 0, err
		}
		err = insertUserAuth(tx, userID, authProvider, authUserID)
	} else {
		err = updateUserFullName(tx, userID, fullName)
	}
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return userID, err
}
