package model

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type DocumentEntry struct {
	RowNumber     int
	AccountID     int
	IsDebit       bool
	UnitCount     int64
	UnitCostCents int64
	Amount        string
	Description   string
}

type Document struct {
	DocumentID      string
	PrevDocumentID  string
	NextDocumentID  string
	HasPrevDocument bool
	HasNextDocument bool
	PaidDateISO     string
	PaidDateFi      string
	Description     string
	PaidUser        User
	CreditAccountID string
	DebitAccountID  string
	ImageID         string
	Amount          string
	AmountCents     int64
	Images          []map[string]string
	Entries         []DocumentEntry
}

func withPaidUser(document sq.SelectBuilder) sq.SelectBuilder {
	return document.LeftJoin(
		"user as paid_user on (paid_user.user_id = document.paid_user_id)").
		Columns("paid_user_id", "paid_user.full_name as paid_user_full_name")
}

func withCents(document sq.SelectBuilder) sq.SelectBuilder {
	sums := sq.Select("unit_count * unit_cost_cents as sum").
		From("document_entry").
		Where("document_entry.document_id = document.document_id")
	debit, _, _ := sums.Where("debit = 1").ToSql()
	credit, _, _ := sums.Where("debit = 0").ToSql()
	return document.Column(fmt.Sprintf("max((%s), (%s)) as cents", debit, credit))
}

func selectDocument() sq.SelectBuilder {
	q := sq.Select("document_id, description, paid_date").
		From("document").OrderBy("document_id, description")
	q = withPaidUser(q)
	q = withCents(q)
	return q
}

func scanDocument(rows sq.RowScanner) (Document, error) {
	var b Document
	var description sql.NullString
	var paidDateISO sql.NullString
	var paidUserID sql.NullInt64
	var paidUserFullName sql.NullString
	var cents sql.NullInt64
	if err := rows.Scan(&b.DocumentID, &description, &paidDateISO,
		&paidUserID, &paidUserFullName, &cents); err != nil {
		return b, err
	}
	b.Description = description.String
	b.PaidDateISO = paidDateISO.String
	b.PaidDateFi = fiFromISODate(b.PaidDateISO)
	b.PaidUser.UserID = paidUserID.Int64
	b.PaidUser.FullName = paidUserFullName.String
	b.AmountCents = cents.Int64
	b.Amount = amountFromCents(cents.Int64)
	return b, nil
}

func (m *Model) GetDocuments() []Document {
	noDocuments := []Document{}
	documents := noDocuments
	q := selectDocument()
	if !m.user.IsAdmin {
		q = q.Where(sq.Eq{"paid_user_id": m.user.UserID})
	}
	rows, err := q.RunWith(m.tx).Query()
	if m.isErr(err) {
		return noDocuments
	}
	defer rows.Close()
	for rows.Next() {
		document, err := scanDocument(rows)
		if m.isErr(err) {
			return noDocuments
		}
		documents = append(documents, document)
	}
	if m.isErr(rows.Err()) {
		return noDocuments
	}
	return documents
}

func (m *Model) GetDocumentsForImages() ([]map[string]interface{}, []int) {
	var images []map[string]interface{}
	var missing []int
	if !m.isAdmin() {
		return images, missing
	}
	rows, err := sq.Select("document.document_id, document_image_num, image.image_id, description, image_data").
		From("document").
		LeftJoin("document_image on document_id = document_id").
		LeftJoin("image on image_id = image_id").
		OrderBy("document.document_id, document_image_num").
		RunWith(m.tx).Query()
	if m.isErr(err) {
		return images, missing
	}
	defer rows.Close()
	for rows.Next() {
		var document_id string
		var document_image_num int
		var image_id string
		var description string
		var image_data []byte
		if m.isErr(rows.Scan(&document_id, &document_image_num, &image_id,
			&description, &image_data)) {
			return images, missing
		}
		images = append(images, map[string]interface{}{
			"document_id":        document_id,
			"document_image_num": document_image_num,
			"image_id":           image_id,
			"description":        description,
			"image_data":         image_data,
		})
	}
	m.isErr(rows.Err())
	return images, missing
}

func (m *Model) getDocumentImages(documentID string) []map[string]string {
	noImages := []map[string]string{}
	images := noImages
	rows, err := sq.Select("image_id").From("document_image").
		Where(sq.Eq{"document_id": documentID}).
		OrderBy("document_image_num").RunWith(m.tx).Query()
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

func (m *Model) getRelativeDocumentID(q sq.SelectBuilder) string {
	if !m.user.IsAdmin {
		q = q.Where(sq.Eq{"paid_user_id": m.user.UserID})
	}
	return m.getIntFromDb(q)
}

func (m *Model) getPrevDocumentID(documentID string) string {
	return m.getRelativeDocumentID(
		sq.Select("max(document_id)").From("document").Where(sq.Lt{"document_id": documentID}))
}

func (m *Model) getNextDocumentID(documentID string) string {
	return m.getRelativeDocumentID(
		sq.Select("min(document_id)").From("document").Where(sq.Gt{"document_id": documentID}))
}

func (m *Model) GetDocumentID(documentID string) *Document {
	b, err := scanDocument(selectDocument().Where(sq.Eq{"document_id": documentID}).
		RunWith(m.tx).QueryRow())
	if err != nil {
		return nil
	}
	if !m.isAdminOrUser(b.PaidUser.UserID) {
		return nil
	}
	m.populateOtherDocumentFieldsFromDocumentEntries(&b)
	b.Images = m.getDocumentImages(documentID)
	if len(b.Images) > 0 {
		b.ImageID = b.Images[0]["ImageID"]
	}
	b.PrevDocumentID = m.getPrevDocumentID(documentID)
	b.NextDocumentID = m.getNextDocumentID(documentID)
	b.HasPrevDocument = (b.PrevDocumentID != "")
	b.HasNextDocument = (b.NextDocumentID != "")
	return &b
}

func selectDocumentEntry() sq.SelectBuilder {
	return sq.Select("row_number, account_id, debit, unit_count, unit_cost_cents, description").
		From("document_entry").
		OrderBy("document_id, row_number")
}

func scanDocumentEntry(rows sq.RowScanner) (DocumentEntry, error) {
	e := DocumentEntry{}
	err := rows.Scan(&e.RowNumber, &e.AccountID,
		&e.IsDebit, &e.UnitCount, &e.UnitCostCents, &e.Description)
	e.Amount = amountFromCents(e.UnitCount * e.UnitCostCents)
	return e, err
}

func (m *Model) documentEntriesFromSelect(selectStmt sq.SelectBuilder) []DocumentEntry {
	noEntries := []DocumentEntry{}
	rows, err := selectStmt.RunWith(m.tx).Query()
	if m.isErr(err) {
		return noEntries
	}
	entries := noEntries
	defer rows.Close()
	for rows.Next() {
		entry, err := scanDocumentEntry(rows)
		if m.isErr(err) {
			return noEntries
		}
		entries = append(entries, entry)
	}
	if m.isErr(rows.Err()) {
		return noEntries
	}
	return entries
}

func (m *Model) GetAllDocumentEntries() []DocumentEntry {
	return m.documentEntriesFromSelect(selectDocumentEntry())
}

func (m *Model) populateDocumentEntries(document *Document) {
	document.Entries = m.documentEntriesFromSelect(
		selectDocumentEntry().Where(sq.Eq{"document_id": document.DocumentID}))
}

func (m *Model) populateDocumentEntriesFromOtherDocumentFields(document *Document) {
	unitCostCents, err := centsFromAmount(document.Amount)
	if m.isErr(err) {
		return
	}
	var entries []DocumentEntry
	addEntry := func(accountIDString, description string, isDebit bool) {
		accountID, _ := strconv.Atoi(accountIDString)
		if accountID > 0 {
			entries = append(entries, DocumentEntry{
				RowNumber:     len(entries),
				UnitCount:     1,
				UnitCostCents: unitCostCents,
				AccountID:     accountID,
				Description:   description,
				IsDebit:       isDebit,
			})
		}
	}
	addEntry(document.CreditAccountID, "Credit", false)
	addEntry(document.DebitAccountID, "Debet", true)
	document.Entries = entries
}

func (m *Model) populateOtherDocumentFieldsFromDocumentEntries(document *Document) {
	q := sq.Select("account_id").From("document_entry").
		Where(sq.Eq{"document_id": document.DocumentID}).
		OrderBy("document_id, row_number")
	q.Where("debit = 0").RunWith(m.tx).Limit(1).
		QueryRow().Scan(&document.CreditAccountID)
	q.Where("debit = 1").RunWith(m.tx).Limit(1).
		QueryRow().Scan(&document.DebitAccountID)
}

func (m *Model) putDocumentEntries(document Document) {
	_, err := sq.Delete("document_entry").Where(sq.Eq{"document_id": document.DocumentID}).
		RunWith(m.tx).Exec()
	if m.isErr(err) {
		return
	}
	for rowNumber, entry := range document.Entries {
		if entry.RowNumber != rowNumber {
			panic("Row number mismatch")
		}
		_, err := sq.Insert("document_entry").SetMap(sq.Eq{
			"document_id":     document.DocumentID,
			"row_number":      rowNumber,
			"unit_count":      1,
			"unit_cost_cents": entry.UnitCostCents,
			"account_id":      entry.AccountID,
			"debit":           entry.IsDebit,
			"description":     entry.Description,
		}).RunWith(m.tx).Exec()
		if m.isErr(err) {
			return
		}
	}
}

func (m *Model) putDocumentImages(document Document) {
	_, err := sq.Delete("document_image").Where(sq.Eq{"document_id": document.DocumentID}).
		RunWith(m.tx).Exec()
	if m.isErr(err) {
		return
	}
	if document.ImageID == "" {
		return
	}
	_, err = sq.Insert("document_image").SetMap(sq.Eq{
		"document_id":        document.DocumentID,
		"document_image_num": 1,
		"image_id":           document.ImageID,
	}).RunWith(m.tx).Exec()
	if m.isErr(err) {
		return
	}
}

func (m *Model) PutDocument(document Document) {
	documentID := parsePositiveInt("document ID", document.DocumentID)
	if documentID < 1 {
		return
	}
	setmap := sq.Eq{
		"description": document.Description,
		"paid_date":   isoFromFiDate(document.PaidDateFi),
	}
	if !m.user.IsAdmin && document.PaidUser.UserID != 0 {
		panic("Non-null PaidUser.UserID for non-admin in PutDocument")
	}
	var oldPaidUserID sql.NullInt64
	if m.isErr(sq.Select("paid_user_id").
		From("document").Where(sq.Eq{"document_id": documentID}).
		RunWith(m.tx).QueryRow().Scan(&oldPaidUserID)) {
		return
	}
	if !m.isAdminOrUser(oldPaidUserID.Int64) {
		return
	}
	if m.user.IsAdmin {
		if document.PaidUser.UserID == 0 {
			setmap["paid_user_id"] = nil
		} else {
			setmap["paid_user_id"] = document.PaidUser.UserID
		}
		m.populateDocumentEntriesFromOtherDocumentFields(&document)
		m.putDocumentEntries(document)
	}
	m.putDocumentImages(document)
	q := sq.Update("document").SetMap(setmap).
		Where(sq.Eq{"document_id": documentID}).
		RunWith(m.tx)
	_, err := q.Exec()
	if m.isErr(err) {
		return
	}
}

func (m *Model) getNewDocumentID() (documentID int64, err error) {
	err = sq.Select("coalesce(max(document_id), 0) + 1").From("document").
		RunWith(m.tx).Limit(1).QueryRow().Scan(&documentID)
	return
}

func (m *Model) PostDocument(document Document) string {
	createdDate := time.Now().Format("2006-01-02")
	documentID, err := m.getNewDocumentID()
	if m.isErr(err) {
		return ""
	}
	setMap := sq.Eq{"document_id": documentID, "created_date": createdDate}
	if !m.user.IsAdmin {
		setMap["paid_user_id"] = m.user.UserID
	}
	_, err = sq.Insert("document").SetMap(setMap).RunWith(m.tx).Exec()
	if m.isErr(err) {
		return ""
	}
	document.DocumentID = strconv.Itoa(int(documentID))
	log.Printf("Created document #%s", document.DocumentID)
	m.PutDocument(document)
	return document.DocumentID
}

type DocumentComp struct {
	DocumentID  string
	Date        string
	Cents       int64
	Description string
}

func (m *Model) GetDocumentsForCompare() []DocumentComp {
	noDocuments := []DocumentComp{}
	if !m.user.IsAdmin {
		m.Forbidden()
		return noDocuments
	}
	documents := noDocuments
	rows, err := selectDocument().RunWith(m.tx).Query()
	if m.isErr(err) {
		return noDocuments
	}
	defer rows.Close()
	for rows.Next() {
		document, err := scanDocument(rows)
		if m.isErr(err) {
			return noDocuments
		}
		documents = append(documents, DocumentComp{
			Date:        document.PaidDateFi,
			Cents:       document.AmountCents,
			Description: document.Description,
		})
	}
	if m.isErr(rows.Err()) {
		return noDocuments
	}
	return documents

}
