package model

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	sq "github.com/Masterminds/squirrel"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
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

func GetBills() interface{} {
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
		paid_date_fi := FiFromIsoDate(paid_date.String)
		closed_date_fi := FiFromIsoDate(closed_date.String)
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

func GetBillsForImages() ([]map[string]interface{}, []int) {
	rows, err := sq.Select("bill.bill_id, bill_image_num, image.image_id, description, image_data").
		From("bill").
		LeftJoin("bill_image on bill_id = bill_id").
		LeftJoin("image on image_id = image_id").
		OrderBy("bill.bill_id, bill_image_num").
		RunWith(db).Query()
	check(err)
	defer rows.Close()
	var images []map[string]interface{}
	var missing []int
	for rows.Next() {
		var bill_id string
		var bill_image_num int
		var image_id string
		var description string
		var image_data []byte
		check(rows.Scan(&bill_id, &bill_image_num, &image_id,
			&description, &image_data))
		images = append(images, map[string]interface{}{
			"bill_id":        bill_id,
			"bill_image_num": bill_image_num,
			"image_id":       image_id,
			"description":    description,
			"image_data":     image_data,
		})
	}
	return images, missing
}

func GetBillImages(billId string) []map[string]string {
	rows, err := sq.Select("image_id").From("bill_image").
		Where(sq.Eq{"bill_id": billId}).
		OrderBy("bill_image_num").RunWith(db).Query()
	check(err)
	defer rows.Close()
	var images []map[string]string
	for rows.Next() {
		var image_id string
		check(rows.Scan(&image_id))
		images = append(images, map[string]string{"image_id": image_id})
	}
	return images
}

func GetBillId(billId string) (map[string]interface{}, error) {
	var bill_id int
	var paid_date sql.NullString
	var closed_date sql.NullString
	var description string
	var paid_user_id int
	var paid_user_full_name string
	q := sq.Select("bill_id, paid_date, closed_date, description").
		From("bill").Where(sq.Eq{"bill_id": billId})
	q = withPaidUser(q)
	q = withCents(q)
	check(q.RunWith(db).Limit(1).QueryRow().Scan(
		&bill_id, &paid_date, &closed_date,
		&description, &paid_user_id, &paid_user_full_name))
	paid_date_fi := FiFromIsoDate(paid_date.String)
	closed_date_fi := FiFromIsoDate(closed_date.String)
	return map[string]interface{}{
		"bill_id":             bill_id,
		"paid_date":           paid_date,
		"paid_date_fi":        paid_date_fi,
		"closed_date":         closed_date,
		"closed_date_fi":      closed_date_fi,
		"description":         description,
		"paid_user_id":        paid_user_id,
		"paid_user_full_name": paid_user_full_name,
		"images":              GetBillImages(billId),
	}, nil
}

func PutBillId(billId string, r *http.Request) error {
	return errors.New("foo")
}

func PostBill(r *http.Request) (string, error) {
	return "", errors.New("foo")
}
