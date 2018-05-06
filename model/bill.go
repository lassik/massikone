package model

import (
	"database/sql"
	"log"
	"strconv"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type BillEntry struct {
	RowNumber     int
	AccountID     int
	IsDebit       bool
	UnitCount     int
	UnitCostCents int
	Description   string
}

type Bill struct {
	BillID          string
	PrevBillID      string
	NextBillID      string
	PaidDateFi      string
	ClosedDateFi    string
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

func (m *Model) GetBills() interface{} {
	var bills []map[string]interface{}
	q := sq.Select("bill_id, paid_date, closed_date, description").
		From("bill").OrderBy("bill_id, description")
	q = withPaidUser(q)
	q = withCents(q)
	if !m.user.IsAdmin {
		q = q.Where(sq.Eq{"paid_user_id": m.user.UserID})
	}
	rows, err := q.RunWith(m.tx).Query()
	if m.isErr(err) {
		return bills
	}
	defer rows.Close()
	for rows.Next() {
		var bill_id int
		var paid_date sql.NullString
		var closed_date sql.NullString
		var description string
		var paid_user_id int
		var paid_user_full_name string
		if m.isErr(rows.Scan(&bill_id, &paid_date, &closed_date,
			&description, &paid_user_id, &paid_user_full_name)) {
			return bills
		}
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
	var images []map[string]string
	rows, err := sq.Select("image_id").From("bill_image").
		Where(sq.Eq{"bill_id": billID}).
		OrderBy("bill_image_num").RunWith(m.tx).Query()
	if m.isErr(err) {
		return images
	}
	defer rows.Close()
	for rows.Next() {
		var image_id string
		if m.isErr(rows.Scan(&image_id)) {
			return images
		}
		images = append(images, map[string]string{"image_id": image_id})
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
	var paidDate sql.NullString
	var closedDate sql.NullString
	q := sq.Select("bill_id, paid_date, closed_date, description").
		From("bill").Where(sq.Eq{"bill_id": billID})
	q = withPaidUser(q)
	q = withCents(q)
	if m.isErr(q.RunWith(m.tx).Limit(1).QueryRow().Scan(
		&bill.BillID, &paidDate, &closedDate,
		&bill.Description, &bill.PaidUser.UserID, &bill.PaidUser.FullName)) {
		return nil
	}
	if !m.isAdminOrUser(bill.PaidUser.UserID) {
		return nil
	}
	bill.PaidDateFi = FiFromIsoDate(paidDate.String)
	bill.ClosedDateFi = FiFromIsoDate(closedDate.String)
	bill.Images = m.getBillImages(billID)
	bill.PrevBillID = m.getPrevBillID(billID)
	bill.NextBillID = m.getNextBillID(billID)
	log.Printf("PrevBillID = %s", bill.PrevBillID)
	log.Printf("NextBillID = %s", bill.NextBillID)
	return &bill
}

func (m *Model) populateBillEntriesFromBillFields(bill Bill) {
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
		"paid_date":   IsoFromFiDate(bill.PaidDateFi),
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
		m.populateBillEntriesFromBillFields(bill)
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
