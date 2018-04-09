package main

import (
	"bytes"
	"database/sql"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/mattn/go-sqlite3"
	"github.com/xo/dburl"
)

const AccountNestingLevel = 9

var databaseUrl = os.Getenv("DATABASE_URL")
var db *sql.DB

func init() {
	var err error
	db, err = dburl.Open(databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
}

func ModelGetUserImageRotated(imageId string) (string, error) {
	imageData, _, err := ModelGetUserImage(imageId)
	if err != nil {
		log.Print(err)
		return "", err
	}
	reader := bytes.NewReader(imageData)
	newImageId, newImageData, err := ImageRotate(reader)
	if err != nil {
		log.Print(err)
		return "", err
	}
	return modelStoreUserImage(newImageId, newImageData)
}

func ModelGetUserImage(imageId string) ([]byte, string, error) {
	var imageData []byte
	check(sq.Select("image_data").From("image").
		Where(sq.Eq{"image_id": imageId}).
		RunWith(db).Limit(1).QueryRow().Scan(&imageData))
	imageMimeType := mime.TypeByExtension(path.Ext(imageId))
	return imageData, imageMimeType, nil
}

func modelStoreUserImage(imageId string, imageData []byte) (string, error) {
	transaction, err := db.Begin()
	if err != nil {
		log.Print(err)
		return "", err
	}
	statement, err := transaction.Prepare(
		"update image set image_id = ?, image_data = ? where image_id = ?")
	if err != nil {
		log.Print(err)
		return "", err
	}
	result, err := statement.Exec(imageId, imageData, imageId)
	if err != nil {
		log.Print(err)
		return "", err
	}
	count, err := result.RowsAffected()
	if err != nil {
		log.Print(err)
		return "", err
	}
	statement.Close()
	if count > 0 {
		transaction.Commit()
		return imageId, nil
	}
	statement, err = transaction.Prepare(
		"insert into image (image_id, image_data) values (?, ?)")
	if err != nil {
		log.Print(err)
		return "", err
	}
	_, err = statement.Exec(imageId, imageData)
	if err != nil {
		log.Print(err)
		return "", err
	}
	statement.Close()
	transaction.Commit()
	return imageId, nil
}

func ModelPostUserImage(reader io.Reader) (string, error) {
	imageId, imageData, err := ImagePrepare(reader)
	if err != nil {
		log.Print(err)
		return "", err
	}
	return modelStoreUserImage(imageId, imageData)
}

func ModelGetAccounts(usedOnly bool) []map[string]interface{} {
	//populateAccounts()
	rows, err := sq.Select("account_id, title, nesting_level").
		From("period_account").OrderBy("account_id, nesting_level").
		RunWith(db).Query()
	check(err)
	defer rows.Close()
	var accounts []map[string]interface{}
	for rows.Next() {
		var account_id string
		var title string
		var nesting_level int
		check(rows.Scan(&account_id, &title, &nesting_level))
		is_account := (nesting_level == AccountNestingLevel)
		dash_level, htag_level := 0, ""
		if !is_account {
			dash_level = 1 + nesting_level
			htag_level = strconv.Itoa(2 + nesting_level)
		}
		prefix := account_id
		if !is_account {
			prefix = strings.Repeat("=", dash_level)
		}
		account_id_or_nil := ""
		if is_account {
			account_id_or_nil = account_id
		}
		accounts = append(accounts,
			map[string]interface{}{
				"raw_account_id": account_id,
				"account_id":     account_id_or_nil,
				"title":          title,
				"prefix":         prefix,
				"htag_level":     htag_level,
			})
	}
	if usedOnly {
		//accounts = reject_unused_accounts(accounts, @db[:bill_entry].select(:account_id).order(:account_id).distinct.map(:account_id))
	}
	return accounts
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

func modelGetBillsForImages() ([]map[string]interface{}, []int) {
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

func modelGetBillImages(billId string) []map[string]string {
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

func ModelGetBillId(billId string) (map[string]interface{}, error) {
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
		"images":              modelGetBillImages(billId),
	}, nil
}

func ModelPutBillId(billId string, r *http.Request) error {
	return errors.New("foo")
}

func ModelPostBill(r *http.Request) (string, error) {
	return "", errors.New("foo")
}
