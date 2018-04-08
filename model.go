package main

import (
	"database/sql"
	"log"
	"os"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/mattn/go-sqlite3"
	"github.com/xo/dburl"
)

var databaseUrl = os.Getenv("DATABASE_URL")
var db *sql.DB

func init() {
	var err error
	db, err = dburl.Open(databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
}

func createGreetings() {
	_, err := db.Exec(
		"create table if not exists greetings (greeting text)",
		nil,
	)
	check(err)
}

func insertGreeting(greeting string) {
	transaction, err := db.Begin()
	check(err)
	statement, err := transaction.Prepare(
		"insert into greetings (greeting) values (?)",
	)
	check(err)
	defer statement.Close()
	_, err = statement.Exec(greeting)
	check(err)
	transaction.Commit()
}

func withPaidUser(bill sq.SelectBuilder) sq.SelectBuilder {
	return bill.LeftJoin("user as paid_user on (paid_user.user_id = bill.paid_user_id)").
		Columns("paid_user_id", "paid_user.full_name as paid_user_full_name")
}

func withCents(bill sq.SelectBuilder) sq.SelectBuilder {
	return bill
	// return {
	// 	sums := sq.Select("unit_count * unit_cost_cents as sum").
	// 		From("bill_entry").
	// 		Where("bill_entry.bill_id = bill.bill_id")
	// 	debit := sums.where("debit")
	// 	credit := sums.exclude("debit")
	// 	max(debit, credit).as("cents")
	// }
}

func ModelGetBills() interface{} {
	q := sq.Select("bill_id, paid_date, closed_date, description").
		From("bill").OrderBy("bill_id, description")
	q = withPaidUser(q)
	q = withCents(q)
	rows, err := q.RunWith(db).Query()
	check(err)
	defer rows.Close()
	var bills []map[string]interface{}
	for rows.Next() {
		var bill_id int
		var paid_date sql.NullString
		var closed_date sql.NullString
		var description string
		var paid_user_id int
		var paid_user_full_name string
		check(rows.Scan(&bill_id, &paid_date, &closed_date,
			&description, &paid_user_id, &paid_user_full_name))
		paid_date_fi := fi_from_iso_date(paid_date.String)
		closed_date_fi := fi_from_iso_date(closed_date.String)
		bills = append(bills, map[string]interface{}{
			"bill_id":             bill_id,
			"paid_date":           paid_date,
			"paid_date_fi":        paid_date_fi,
			"closed_date":         closed_date,
			"closed_date_fi":      closed_date_fi,
			"description":         description,
			"paid_user_id":        paid_user_id,
			"paid_user_full_name": paid_user_full_name,
		})
	}
	check(rows.Err())
	return bills
}

func ModelGetImageData(imageId string) ([]byte, error) {
	//raise unless valid_image_id?(imageId)
	// TODO: what if not found
	rows, err := sq.Select("image_data").From("image").
		Where(sq.Eq{"image_id": imageId}).RunWith(db).Limit(1).Query()
	if err != nil {
		return []byte{}, err
	}
	var imageData []byte
	defer rows.Close()
	for rows.Next() {
		check(rows.Scan(&imageData))
	}
	check(rows.Err())
	return imageData, nil
}
