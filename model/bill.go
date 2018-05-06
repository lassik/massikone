package model

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type BillEntry struct {
	RowNumber     int
	AccountID     int
	IsDebit       bool
	UnitCount     int64
	UnitCostCents int64
	Description   string
}

type Bill struct {
	BillID          string
	PrevBillID      string
	NextBillID      string
	HasPrevBill     bool
	HasNextBill     bool
	PaidDateISO     string
	PaidDateFi      string
	Description     string
	PaidUser        User
	CreditAccountID string
	DebitAccountID  string
	ImageID         string
	Amount          string
	Images          []map[string]string
	Entries         []BillEntry
}

func withPaidUser(bill sq.SelectBuilder) sq.SelectBuilder {
	return bill.LeftJoin(
		"user as paid_user on (paid_user.user_id = bill.paid_user_id)").
		Columns("paid_user_id", "paid_user.full_name as paid_user_full_name")
}

func withCents(bill sq.SelectBuilder) sq.SelectBuilder {
	sums := sq.Select("unit_count * unit_cost_cents as sum").
		From("bill_entry").
		Where("bill_entry.bill_id = bill.bill_id")
	debit, _, _ := sums.Where("debit = 1").ToSql()
	credit, _, _ := sums.Where("debit = 0").ToSql()
	return bill.Column(fmt.Sprintf("max((%s), (%s)) as cents", debit, credit))
}

func (m *Model) GetBills() []Bill {
	var bills []Bill
	q := sq.Select("bill_id, paid_date, description").
		From("bill").OrderBy("bill_id, description")
	q = withPaidUser(q)
	sqlString, _, _ := q.ToSql()
	log.Print(sqlString)
	q = withCents(q)
	sqlString, _, _ = q.ToSql()
	log.Print(sqlString)
	if !m.user.IsAdmin {
		q = q.Where(sq.Eq{"paid_user_id": m.user.UserID})
	}
	rows, err := q.RunWith(m.tx).Query()
	if m.isErr(err) {
		return bills
	}
	defer rows.Close()
	for rows.Next() {
		var b Bill
		var paidDateISO sql.NullString
		var cents sql.NullInt64
		if m.isErr(rows.Scan(&b.BillID, &paidDateISO, &b.Description,
			&b.PaidUser.UserID, &b.PaidUser.FullName, &cents)) {
			return bills
		}
		b.PaidDateISO = paidDateISO.String
		b.PaidDateFi = fiFromISODate(paidDateISO.String)
		b.Amount = amountFromCents(cents.Int64)
		bills = append(bills, b)
	}
	if m.isErr(rows.Err()) {
		return bills
	}
	return bills
}

func (m *Model) GetBillsForImages() ([]map[string]interface{}, []int) {
	var images []map[string]interface{}
	var missing []int
	if !m.isAdmin() {
		return images, missing
	}
	rows, err := sq.Select("bill.bill_id, bill_image_num, image.image_id, description, image_data").
		From("bill").
		LeftJoin("bill_image on bill_id = bill_id").
		LeftJoin("image on image_id = image_id").
		OrderBy("bill.bill_id, bill_image_num").
		RunWith(m.tx).Query()
	if m.isErr(err) {
		return images, missing
	}
	defer rows.Close()
	for rows.Next() {
		var bill_id string
		var bill_image_num int
		var image_id string
		var description string
		var image_data []byte
		if m.isErr(rows.Scan(&bill_id, &bill_image_num, &image_id,
			&description, &image_data)) {
			return images, missing
		}
		images = append(images, map[string]interface{}{
			"bill_id":        bill_id,
			"bill_image_num": bill_image_num,
			"image_id":       image_id,
			"description":    description,
			"image_data":     image_data,
		})
	}
	m.isErr(rows.Err())
	return images, missing
}

func (m *Model) getBillImages(billID string) []map[string]string {
	noImages := []map[string]string{}
	images := noImages
	rows, err := sq.Select("image_id").From("bill_image").
		Where(sq.Eq{"bill_id": billID}).
		OrderBy("bill_image_num").RunWith(m.tx).Query()
	if m.isErr(err) {
		return noImages
	}
	defer rows.Close()
	for rows.Next() {
		var imageID string
		if m.isErr(rows.Scan(&imageID)) {
			return noImages
		}
		thisImage := map[string]string{"ImageID": imageID}
		images = append(images, thisImage)
	}
	return images
}

func (m *Model) getRelativeBillID(q sq.SelectBuilder) string {
	if !m.user.IsAdmin {
		q = q.Where(sq.Eq{"paid_user_id": m.user.UserID})
	}
	return m.getIntFromDb(q)
}

func (m *Model) getPrevBillID(billID string) string {
	return m.getRelativeBillID(
		sq.Select("max(bill_id)").From("bill").Where(sq.Lt{"bill_id": billID}))
}

func (m *Model) getNextBillID(billID string) string {
	return m.getRelativeBillID(
		sq.Select("min(bill_id)").From("bill").Where(sq.Gt{"bill_id": billID}))
}

func (m *Model) GetBillID(billID string) *Bill {
	var bill Bill
	var paidDateISO sql.NullString
	var cents sql.NullInt64
	q := sq.Select("bill_id, paid_date, description").
		From("bill").Where(sq.Eq{"bill_id": billID})
	q = withPaidUser(q)
	q = withCents(q)
	if m.isErr(q.RunWith(m.tx).Limit(1).QueryRow().Scan(
		&bill.BillID, &paidDateISO, &bill.Description,
		&bill.PaidUser.UserID,
		&bill.PaidUser.FullName, &cents)) {
		return nil
	}
	if !m.isAdminOrUser(bill.PaidUser.UserID) {
		return nil
	}
	bill.PaidDateISO = paidDateISO.String
	bill.PaidDateFi = fiFromISODate(paidDateISO.String)
	bill.Amount = amountFromCents(cents.Int64)
	m.populateOtherBillFieldsFromBillEntries(&bill)
	bill.Images = m.getBillImages(billID)
	if len(bill.Images) > 0 {
		bill.ImageID = bill.Images[0]["ImageID"]
	}
	bill.PrevBillID = m.getPrevBillID(billID)
	bill.NextBillID = m.getNextBillID(billID)
	bill.HasPrevBill = (bill.PrevBillID != "")
	bill.HasNextBill = (bill.NextBillID != "")
	return &bill
}

func (m *Model) populateBillEntriesFromOtherBillFields(bill *Bill) {
	unitCostCents, err := centsFromAmount(bill.Amount)
	if m.isErr(err) {
		return
	}
	var entries []BillEntry
	addEntry := func(accountIDString, description string, isDebit bool) {
		accountID, _ := strconv.Atoi(bill.CreditAccountID)
		if accountID > 0 {
			entries = append(entries, BillEntry{
				RowNumber:     len(entries),
				UnitCount:     1,
				UnitCostCents: unitCostCents,
				AccountID:     accountID,
				Description:   description,
				IsDebit:       isDebit,
			})
		}
	}
	addEntry(bill.CreditAccountID, "Credit", false)
	addEntry(bill.DebitAccountID, "Debet", true)
	bill.Entries = entries
}

func (m *Model) populateOtherBillFieldsFromBillEntries(bill *Bill) {
	q := sq.Select("account_id").From("bill_entry").
		Where(sq.Eq{"bill_id": bill.BillID}).
		OrderBy("bill_id, row_number")
	q.Where("debit = 0").RunWith(m.tx).Limit(1).
		QueryRow().Scan(&bill.CreditAccountID)
	q.Where("debit = 1").RunWith(m.tx).Limit(1).
		QueryRow().Scan(&bill.DebitAccountID)
}

func (m *Model) putBillEntries(bill Bill) {
	_, err := sq.Delete("bill_entry").Where(sq.Eq{"bill_id": bill.BillID}).
		RunWith(m.tx).Exec()
	if m.isErr(err) {
		return
	}
	for rowNumber, entry := range bill.Entries {
		if entry.RowNumber != rowNumber {
			panic("Row number mismatch")
		}
		if m.isErr(sq.Insert("bill_entry").
			SetMap(sq.Eq{
				"bill_id":         bill.BillID,
				"row_number":      rowNumber,
				"unit_count":      1,
				"unit_cost_cents": entry.UnitCostCents,
				"account_id":      entry.AccountID,
				"debit":           entry.IsDebit,
				"description":     entry.Description,
			}).RunWith(m.tx).QueryRow().Scan()) {
			return
		}
	}
}

func (m *Model) putBillImages(bill Bill) {
	_, err := sq.Delete("bill_image").Where(sq.Eq{"bill_id": bill.BillID}).
		RunWith(m.tx).Exec()
	if m.isErr(err) {
		return
	}
	if bill.ImageID == "" {
		return
	}
	if m.isErr(sq.Insert("bill_image").
		SetMap(sq.Eq{
			"bill_id":        bill.BillID,
			"bill_image_num": 1,
			"image_id":       bill.ImageID,
		}).RunWith(m.tx).QueryRow().Scan(&bill.BillID)) {
		return
	}
}

func (m *Model) PutBill(bill Bill) {
	if bill.BillID == "" {
		panic("Null BillID in PutBill")
	}
	setmap := sq.Eq{
		"description": bill.Description,
		"paid_date":   isoFromFiDate(bill.PaidDateFi),
	}
	if !m.user.IsAdmin && bill.PaidUser.UserID != "" {
		panic("Non-null PaidUser.UserID for non-admin in PutBill")
	}
	var oldPaidUserID string
	if m.isErr(sq.Select("paid_user_id").
		From("bill").Where(sq.Eq{"bill_id": bill.BillID}).
		RunWith(m.tx).QueryRow().Scan(&oldPaidUserID)) {
		return
	}
	if !m.isAdminOrUser(oldPaidUserID) {
		return
	}
	if m.user.IsAdmin {
		m.populateBillEntriesFromOtherBillFields(&bill)
		m.putBillEntries(bill)
	}
	m.putBillImages(bill)
	//"paid_user_id": bill.PaidUser.UserID
	m.isErr(sq.Update("bill").SetMap(setmap).
		Where(sq.Eq{"bill_id": bill.BillID}).
		RunWith(m.tx).QueryRow().Scan())
}

func (m *Model) PostBill(bill Bill) string {
	createdDate := time.Now().Format("2006-01-02")
	if m.isErr(sq.Insert("bill").
		SetMap(sq.Eq{"created_date": createdDate}).
		RunWith(m.tx).QueryRow().Scan(&bill.BillID)) {
		return ""
	}
	m.PutBill(bill)
	return bill.BillID
}
