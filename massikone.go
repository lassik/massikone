package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gobuffalo/packr"
	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	"github.com/hoisie/mustache"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/gplus"

	"./model"
	"./reports"
)

const sessionCurrentUser = "current_user"

var sessionName = os.Getenv("SESSION_NAME")
var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
var port = os.Getenv("PORT")
var staticBox = packr.NewBox("./public")
var templatesBox = packr.NewBox("./views")

func templateFromBox(filename string) *mustache.Template {
	tmplString, err := templatesBox.MustString(filename)
	if err != nil {
		panic(err)
	}
	tmpl, err := mustache.ParseString(tmplString)
	if err != nil {
		panic(err)
	}
	return tmpl
}

var billsTemplate = templateFromBox("bills.mustache")
var billTemplate = templateFromBox("bill.mustache")
var compareTemplate = templateFromBox("compare.mustache")
var loginTemplate = templateFromBox("login.mustache")

func init() {
	baseURL := "http://127.0.0.1:" + port
	goth.UseProviders(
		gplus.New(
			os.Getenv("GPLUS_KEY"),
			os.Getenv("GPLUS_SECRET"),
			baseURL+"/auth/gplus/callback"),
	)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getAppTitle() string {
	organization := "Testi"
	return organization + " Massikone"
}

func setCurrentUserId(w http.ResponseWriter, r *http.Request, id string) {
	session, _ := store.Get(r, sessionName)
	if id == "" {
		delete(session.Values, sessionCurrentUser)
	} else {
		session.Values[sessionCurrentUser] = id
	}
	session.Save(r, w)
}

func getCurrentUserId(r *http.Request) string {
	session, _ := store.Get(r, sessionName)
	if id, ok := session.Values[sessionCurrentUser]; ok {
		if sid, ok := id.(string); ok {
			return sid
		}
	}
	return ""
}

func authCallbackHandler(w http.ResponseWriter, r *http.Request) {
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		log.Print(err)
		fmt.Fprintln(w, err)
		return
	}
	setCurrentUserId(w, r, user.Name)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func postLogout(w http.ResponseWriter, r *http.Request) {
	setCurrentUserId(w, r, "")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getBillsOrLogin(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUserId(r)
	if user == "" {
		w.Write([]byte(loginTemplate.Render(
			map[string]string{"app_title": getAppTitle()})))
		return
	}
	bills := model.GetBills()
	w.Write([]byte(billsTemplate.Render(
		map[string]interface{}{
			"app_title": getAppTitle(),
			"current_user": map[string]string{
				"full_name": user,
			},
			"bills": map[string]interface{}{
				"bills": bills,
			},
		})))
}

func getBillId(w http.ResponseWriter, r *http.Request) {
	//accounts := model.GetAccounts(false)
	billId := r.URL.Query().Get(":billId")
	bill, err := model.GetBillId(billId)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	w.Write([]byte(billTemplate.Render(
		map[string]interface{}{
			"app_title": getAppTitle(),
			"bill":      bill,
		})))
}

func putBillId(w http.ResponseWriter, r *http.Request) {
	billId := r.URL.Query().Get(":billId")
	err := model.PutBillId(billId, r)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/bill/"+billId, http.StatusSeeOther)
}

func getBill(w http.ResponseWriter, r *http.Request) {
	// accounts = model.get_accounts
	w.Write([]byte(billTemplate.Render(
		map[string]string{
			"app_title": getAppTitle(),
			// current_user: model.user,
			// admin: admin_data,
			// credit_accounts: accounts,
			// debit_accounts: accounts
		})))
}

func postBill(w http.ResponseWriter, r *http.Request) {
	billId, err := model.PostBill(r)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/bill/"+billId, http.StatusSeeOther)
}

func getCompare(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(compareTemplate.Render(
		map[string]string{"app_title": getAppTitle()})))
}

func getImageRotated(w http.ResponseWriter, r *http.Request) {
	imageId := r.URL.Query().Get(":imageId")
	rotatedImageId, err := model.GetImageRotated(imageId)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(rotatedImageId))
}

// TODO: http header, esp. caching
func getImage(w http.ResponseWriter, r *http.Request) {
	imageId := r.URL.Query().Get(":imageId")
	imageData, imageMimeType, err := model.GetImage(imageId)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", imageMimeType)
	w.Write(imageData)
}

func postImage(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer file.Close()
	imageId, err := model.PostImage(file)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(imageId))
}

func report(generate func(reports.GetWriter)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		generate(func(mimeType, filename string) (io.Writer, error) {
			w.Header().Set("Content-Type", mimeType)
			w.Header().Set("Content-Disposition",
				fmt.Sprintf("attachment; filename=%q", filename))
			return w, nil
		})
	}
}

func allUsers(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		userId := getCurrentUserId(r)
		if userId != "" {
			handler(w, r)
			return
		}
		http.Error(w, http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized)
	}
}

func adminOnly(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		isAdmin := true
		if isAdmin {
			handler(w, r)
			return
		}
		http.Error(w, http.StatusText(http.StatusUnauthorized),
			http.StatusUnauthorized)
	}
}

func main() {
	p := pat.New()

	p.Get(`/api/userimage/rotated/{imageId:[0-9a-f]{40}\.(?:jpeg|png)}`,
		allUsers(getImageRotated))
	p.Get(`/api/userimage/{imageId:[0-9a-f]{40}\.(?:jpeg|png)}`,
		allUsers(getImage))
	p.Post(`/api/userimage`,
		allUsers(postImage))
	p.Get(`/bill/{billId:[0-9]+}`,
		allUsers(getBillId))
	p.Put(`/bill/{billId:[0-9]+}`,
		allUsers(putBillId))
	p.Get(`/bill`,
		allUsers(getBill))
	p.Post(`/bill`,
		allUsers(postBill))

	//p.Put(`/api/preferences`,
	//	adminOnly(model.PutPreferences))
	//p.Get(`/api/compare`,
	//	adminOnly(model.GetBillsForCompare))
	p.Get(`/compare`,
		adminOnly(getCompare))
	p.Get(`/report/income-statement`,
		adminOnly(report(reports.IncomeStatementPdf)))
	p.Get(`/report/income-statement-detailed`,
		adminOnly(report(reports.IncomeStatementDetailedPdf)))
	p.Get(`/report/balance-sheet`,
		adminOnly(report(reports.BalanceSheetPdf)))
	p.Get(`/report/balance-sheet-detailed`,
		adminOnly(report(reports.BalanceSheetDetailedPdf)))
	p.Get(`/report/general-journal`,
		adminOnly(report(reports.GeneralJournalPdf)))
	p.Get(`/report/general-ledger`,
		adminOnly(report(reports.GeneralLedgerPdf)))
	p.Get(`/report/chart-of-accounts`,
		adminOnly(report(reports.ChartOfAccountsPdf)))
	p.Get(`/report/full-statement`,
		adminOnly(report(reports.FullStatementZip)))

	p.Get(`/auth/{provider}/callback`, authCallbackHandler)
	p.Get(`/auth/{provider}`, gothic.BeginAuthHandler)
	p.Post(`/logout`, postLogout)
	p.Get(`/`, getBillsOrLogin)

	mux := http.NewServeMux()
	mux.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(staticBox)))
	mux.Handle("/", p)
	log.Print("Starting web server")
	http.ListenAndServe(":"+port, mux)
}
