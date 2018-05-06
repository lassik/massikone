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

func (m *Model) getUserByID(userID string) (*User, error) {
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

func GetOrPutUser(provider, providerUserID, email, fullName string) (string, error) {
	providerUserIDColumn := "user_id_" + provider
	tx, err := db.Begin()
	setmap := sq.Eq{
		providerUserIDColumn: providerUserID,
		"email":              email,
		"full_name":          fullName,
	}
	err = sq.Update("user").SetMap(setmap).RunWith(tx).QueryRow().Scan()
	if err == sql.ErrNoRows {
		var userCount int
		err = sq.Select("count(*)").From("user").
			RunWith(tx).Limit(1).QueryRow().Scan(&userCount)
		if err != nil {
                        return "", err
                }
		isAdmin := (userCount == 0)
		setmap["is_admin"] = isAdmin
		sq.Insert("user").SetMap(setmap).RunWith(tx).QueryRow().Scan()
	} else if err != nil {
                return "", err
        }
	var userID string
	err = sq.Select("user_id").From("user").Where(sq.Eq{providerUserIDColumn: providerUserID}).RunWith(tx).Limit(1).QueryRow().Scan(&userID)
	if err != nil {
                return "", err
        }
	if err = tx.Commit(); err != nil {
                return "", err
        }
	return userID, err
}
