package model

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
)

type User struct {
	UserID   string
	FullName string
	IsAdmin  bool
}

func getUserByID(userID string) *User {
	var user User
	q := sq.Select("user_id, full_name, is_admin").
		From("user").Where(sq.Eq{"user_id": userID})
	err := q.RunWith(db).Limit(1).QueryRow().Scan(
		&user.UserID, &user.FullName, &user.IsAdmin)
	if err == sql.ErrNoRows {
		return nil
	}
	check(err)
	return &user
}

func GetOrPutUser(provider, providerUserID, email, fullName string) string {
	var userID string
	q := sq.Select("user_id").
		From("user").Where(sq.Eq{"user_id_" + provider: providerUserID})
	// email
	// fullName
	check(q.RunWith(db).Limit(1).QueryRow().Scan(&userID))
	return userID
}
