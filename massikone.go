package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
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

func setCurrentUserID(w http.ResponseWriter, r *http.Request, id string) {
	session, _ := store.Get(r, sessionName)
	if id == "" {
		delete(session.Values, sessionCurrentUser)
	} else {
		session.Values[sessionCurrentUser] = id
	}
	session.Save(r, w)
}

func getCurrentUserID(r *http.Request) string {
	session, _ := store.Get(r, sessionName)
	if id, ok := session.Values[sessionCurrentUser]; ok {
		if sid, ok := id.(string); ok {
			return sid
		}
	}
	return ""
}

type ModelHandlerFunc func(*model.Model, http.ResponseWriter, *http.Request)

func withModel(h ModelHandlerFunc, adminOnly bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := model.MakeModel(getCurrentUserID(r), adminOnly)
		if m.Err != nil {
			log.Print(m.Err)
			http.Error(w, http.StatusText(http.StatusUnauthorized),
				http.StatusUnauthorized)
			return
		}
		h(&m, w, r)
	}
}

func anyUser(h ModelHandlerFunc) http.HandlerFunc {
	return withModel(h, false)
}

func adminOnly(h ModelHandlerFunc) http.HandlerFunc {
	return withModel(h, true)
}

func authCallbackHandler(w http.ResponseWriter, r *http.Request) {
	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		log.Print(err)
		return
	}
	provider := map[string]string{"gplus": "google_oauth2"}[gothUser.Provider]
	if provider == "" {
		log.Print("Unknown provider")
		return
	}
	userID := model.GetOrPutUser(
		provider, gothUser.UserID, gothUser.Email, gothUser.Name)
	if err != nil {
		log.Print(err)
		return
	}
	setCurrentUserID(w, r, userID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func postLogout(w http.ResponseWriter, r *http.Request) {
	setCurrentUserID(w, r, "")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getBills(m *model.Model, w http.ResponseWriter, r *http.Request) {
	bills := m.GetBills()
	w.Write([]byte(billsTemplate.Render(
		map[string]interface{}{
			"app_title": getAppTitle(),
			"current_user": map[string]string{
				"full_name": m.User().FullName,
			},
			"bills": map[string]interface{}{
				"bills": bills,
			},
		})))
}

func getBillsOrLogin(w http.ResponseWriter, r *http.Request) {
	userID := getCurrentUserID(r)
	if userID == "" {
		w.Write([]byte(loginTemplate.Render(
			map[string]string{"app_title": getAppTitle()})))
		return
	}
	anyUser(getBills)(w, r)
}

func getBillID(m *model.Model, w http.ResponseWriter, r *http.Request) {
	//accounts := model.GetAccounts(false)
	billID := mux.Vars(r)["billID"]
	bill, err := m.GetBillID(billID)
	if bill == nil {
		http.NotFound(w, r)
		return
	}
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

func putBillID(m *model.Model, w http.ResponseWriter, r *http.Request) {
	billID := r.URL.Query().Get(":billID")
	err := m.PutBillID(billID, r)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/bill/"+billID, http.StatusSeeOther)
}

func getBill(m *model.Model, w http.ResponseWriter, r *http.Request) {
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

func postBill(m *model.Model, w http.ResponseWriter, r *http.Request) {
	billID, err := m.PostBill(r)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/bill/"+billID, http.StatusSeeOther)
}

func getCompare(m *model.Model, w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(compareTemplate.Render(
		map[string]string{"app_title": getAppTitle()})))
}

func getImageRotated(m *model.Model, w http.ResponseWriter, r *http.Request) {
	imageID := r.URL.Query().Get(":imageID")
	rotatedImageID, err := model.GetImageRotated(imageID)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(rotatedImageID))
}

// TODO: http header, esp. caching
func getImage(m *model.Model, w http.ResponseWriter, r *http.Request) {
	imageID := r.URL.Query().Get(":imageID")
	imageData, imageMimeType, err := model.GetImage(imageID)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", imageMimeType)
	w.Write(imageData)
}

func postImage(m *model.Model, w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer file.Close()
	imageID, err := model.PostImage(file)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(imageID))
}

func report(generate func(*model.Model, reports.GetWriter)) ModelHandlerFunc {
	return func(m *model.Model, w http.ResponseWriter, r *http.Request) {
		generate(m, func(mimeType, filename string) (io.Writer, error) {
			w.Header().Set("Content-Type", mimeType)
			w.Header().Set("Content-Disposition",
				fmt.Sprintf("attachment; filename=%q", filename))
			return w, nil
		})
	}
}

func main() {
	var router mux.Router

	get := func(path string, h http.HandlerFunc) {
		router.NewRoute().Path(path).Handler(h).Methods("GET")
	}
	put := func(path string, h http.HandlerFunc) {
		router.NewRoute().Path(path).Handler(h).Methods("PUT")
	}
	post := func(path string, h http.HandlerFunc) {
		router.NewRoute().Path(path).Handler(h).Methods("POST")
	}

	get(`/api/userimage/rotated/{imageID}`,
		anyUser(getImageRotated))
	get(`/api/userimage/{imageID}`,
		anyUser(getImage))
	post(`/api/userimage`,
		anyUser(postImage))
	get(`/bill/{billID}`,
		anyUser(getBillID))
	put(`/bill/{billID}`,
		anyUser(putBillID))
	get(`/bill`,
		anyUser(getBill))
	post(`/bill`,
		anyUser(postBill))

	//p.Put(`/api/preferences`,
	//	adminOnly(model.PutPreferences))
	//get(`/api/compare`,
	//	adminOnly(model.GetBillsForCompare))
	get(`/compare`,
		adminOnly(getCompare))
	get(`/report/income-statement`,
		adminOnly(report(reports.IncomeStatementPdf)))
	get(`/report/income-statement-detailed`,
		adminOnly(report(reports.IncomeStatementDetailedPdf)))
	get(`/report/balance-sheet`,
		adminOnly(report(reports.BalanceSheetPdf)))
	get(`/report/balance-sheet-detailed`,
		adminOnly(report(reports.BalanceSheetDetailedPdf)))
	get(`/report/general-journal`,
		adminOnly(report(reports.GeneralJournalPdf)))
	get(`/report/general-ledger`,
		adminOnly(report(reports.GeneralLedgerPdf)))
	get(`/report/chart-of-accounts`,
		adminOnly(report(reports.ChartOfAccountsPdf)))
	get(`/report/full-statement`,
		adminOnly(report(reports.FullStatementZip)))

	get(`/auth/{provider}/callback`, authCallbackHandler)
	get(`/auth/{provider}`, gothic.BeginAuthHandler)
	get(`/logout`, postLogout)
	post(`/logout`, postLogout)
	get(`/`, getBillsOrLogin)

	http.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(staticBox)))
	http.Handle("/", &router)
	log.Print("Starting web server")
	http.ListenAndServe(":"+port, nil)
}
