package main

//go:generate go run staticgen/main.go

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/hoisie/mustache"
	"github.com/lassik/airfreight"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/gplus"
	"github.com/subosito/gotenv"
	"github.com/toqueteos/webbrowser"

	"github.com/lassik/massikone/model"
	"github.com/lassik/massikone/reports"
)

const sessionName = "massikone"
const sessionCurrentUser = "current_user"

var cookieStore *sessions.CookieStore
var publicURL string

func getTemplate(filename string) *mustache.Template {
	tmplString := templates[filename].Contents
	if tmplString == "" {
		panic("Template not found")
	}
	tmpl, err := mustache.ParseString(tmplString)
	if err != nil {
		panic(err)
	}
	return tmpl
}

var billsTemplate = getTemplate("/bills.mustache")
var billTemplate = getTemplate("/bill.mustache")
var settingsTemplate = getTemplate("/settings.mustache")
var compareTemplate = getTemplate("/compare.mustache")
var loginTemplate = getTemplate("/login.mustache")

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getSessionSecret(fromEnv string) []byte {
	const minLen = 32
	secret := []byte(fromEnv)
	if len(secret) < minLen {
		secret = make([]byte, minLen)
		_, err := rand.Read(secret)
		check(err)
	}
	return secret
}

func getAppTitle(settings model.Settings) string {
	return settings.OrgShortName + " Massikone"
}

func setSessionUserID(w http.ResponseWriter, r *http.Request, id int64) {
	session, _ := cookieStore.Get(r, sessionName)
	if id == 0 {
		delete(session.Values, sessionCurrentUser)
	} else {
		session.Values[sessionCurrentUser] = strconv.FormatInt(id, 10)
	}
	session.Save(r, w)
}

func getSessionUserID(r *http.Request) int64 {
	if publicURL == "" {
		return 1
	}
	session, _ := cookieStore.Get(r, sessionName)
	if id, ok := session.Values[sessionCurrentUser]; ok {
		if sid, ok := id.(string); ok {
			if iid, err := strconv.Atoi(sid); err == nil {
				return int64(iid)
			}
		}
	}
	return 0
}

type ModelHandlerFunc func(*model.Model, http.ResponseWriter, *http.Request)

func withModel(h ModelHandlerFunc, adminOnly bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := model.MakeModel(getSessionUserID(r), adminOnly)
		defer m.Close()
		if m.Err != nil {
			log.Print(m.Err)
			http.Error(w, http.StatusText(http.StatusUnauthorized),
				http.StatusUnauthorized)
			return
		}
		h(&m, w, r)
		if m.Err != nil {
			log.Print(m.Err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
			return
		}
	}
}

func anyUser(h ModelHandlerFunc) http.HandlerFunc {
	return withModel(h, false)
}

func adminOnly(h ModelHandlerFunc) http.HandlerFunc {
	return withModel(h, true)
}

func finishLogin(w http.ResponseWriter, r *http.Request) {
	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		log.Print(err)
		return
	}
	userID, err := model.GetOrPutUser(
		gothUser.Provider, gothUser.UserID, gothUser.Name)
	if err != nil {
		log.Print(err)
		return
	}
	setSessionUserID(w, r, userID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func logout(w http.ResponseWriter, r *http.Request) {
	if publicURL != "" {
		setSessionUserID(w, r, 0)
		gothic.Logout(w, r)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getLoginPage(w http.ResponseWriter, r *http.Request) {
	settings := model.GetSettings()
	w.Write([]byte(loginTemplate.Render(
		map[string]string{"AppTitle": getAppTitle(settings)})))
}

func getBills(m *model.Model, w http.ResponseWriter, r *http.Request) {
	settings := m.GetSettings()
	bills := m.GetBills()
	w.Write([]byte(billsTemplate.Render(
		map[string]interface{}{
			"AppTitle":    getAppTitle(settings),
			"IsPublic":    (publicURL != ""),
			"CurrentUser": m.User(),
			"Bills": map[string][]model.Bill{
				"Bills": bills,
			},
		})))
}

func getBillsOrLogin(w http.ResponseWriter, r *http.Request) {
	if getSessionUserID(r) == 0 {
		getLoginPage(w, r)
	} else {
		anyUser(getBills)(w, r)
	}
}

func getBillID(m *model.Model, w http.ResponseWriter, r *http.Request) {
	settings := m.GetSettings()
	billID := mux.Vars(r)["billID"]
	bill := m.GetBillID(billID)
	if m.Err != nil {
		return
	}
	if bill == nil {
		http.NotFound(w, r)
		return
	}
	var users []model.User
	var creditAccounts []model.Account
	var debitAccounts []model.Account
	if m.User().IsAdmin {
		users = m.GetUsers(bill.PaidUser.UserID)
		creditAccounts = m.GetAccounts(false, bill.CreditAccountID)
		debitAccounts = m.GetAccounts(false, bill.DebitAccountID)
	}
	w.Write([]byte(billTemplate.Render(
		map[string]interface{}{
			"AppTitle":       getAppTitle(settings),
			"CurrentUser":    m.User(),
			"Bill":           bill,
			"Users":          users,
			"CreditAccounts": creditAccounts,
			"DebitAccounts":  debitAccounts,
		})))
}

func billFromRequest(r *http.Request, billID string) model.Bill {
	paidUserID, _ := strconv.Atoi(r.PostFormValue("paid_user_id"))
	return model.Bill{
		BillID:          billID,
		PaidDateFi:      r.PostFormValue("paid_date_fi"),
		Description:     r.PostFormValue("description"),
		ImageID:         r.PostFormValue("image_id"),
		Amount:          r.PostFormValue("amount"),
		CreditAccountID: r.PostFormValue("credit_account_id"),
		DebitAccountID:  r.PostFormValue("debit_account_id"),
		PaidUser: model.User{
			UserID: int64(paidUserID),
		},
	}
}

func putBillID(m *model.Model, w http.ResponseWriter, r *http.Request) {
	billID := mux.Vars(r)["billID"]
	m.PutBill(billFromRequest(r, billID))
	if m.Err != nil {
		return
	}
	http.Redirect(w, r, "/tosite/"+billID, http.StatusSeeOther)
}

func postBill(m *model.Model, w http.ResponseWriter, r *http.Request) {
	billID := m.PostBill(billFromRequest(r, ""))
	if m.Err != nil {
		return
	}
	http.Redirect(w, r, "/tosite/"+billID, http.StatusSeeOther)
}

func getNewBillPage(m *model.Model, w http.ResponseWriter, r *http.Request) {
	settings := m.GetSettings()
	var users []model.User
	var accounts []model.Account
	if m.User().IsAdmin {
		users = m.GetUsers(0)
		accounts = m.GetAccounts(false, "")
	}
	w.Write([]byte(billTemplate.Render(
		map[string]interface{}{
			"AppTitle":       getAppTitle(settings),
			"CurrentUser":    m.User(),
			"Users":          users,
			"CreditAccounts": accounts,
			"DebitAccounts":  accounts,
		})))
}

func getSettings(m *model.Model, w http.ResponseWriter, r *http.Request) {
	settings := m.GetSettings()
	users := m.GetUsers(0)
	w.Write([]byte(settingsTemplate.Render(
		map[string]interface{}{
			"AppTitle":    getAppTitle(settings),
			"CurrentUser": m.User(),
			"Settings":    model.GetSettings(),
			"Users":       users,
		})))
}

func putSettings(m *model.Model, w http.ResponseWriter, r *http.Request) {
	m.PutSettings(model.Settings{
		OrgFullName:  r.PostFormValue("OrgFullName"),
		OrgShortName: r.PostFormValue("OrgShortName"),
	})
	http.Redirect(w, r, "/asetukset", http.StatusSeeOther)
}

func getCompare(m *model.Model, w http.ResponseWriter, r *http.Request) {
	settings := m.GetSettings()
	w.Write([]byte(compareTemplate.Render(
		map[string]string{"AppTitle": getAppTitle(settings)})))
}

func getImageRotated(m *model.Model, w http.ResponseWriter, r *http.Request) {
	imageID := mux.Vars(r)["imageID"]
	rotatedImageID, err := m.GetImageRotated(imageID)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(rotatedImageID))
}

// TODO: http header, esp. caching
func getImage(m *model.Model, w http.ResponseWriter, r *http.Request) {
	imageID := mux.Vars(r)["imageID"]
	imageData, imageMimeType, err := m.GetImage(imageID)
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
	imageID, err := m.PostImage(file)
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
	log.Printf("Massikone (%s, %s/%s)",
		runtime.Version(), runtime.GOOS, runtime.GOARCH)
	wd, _ := os.Getwd()
	log.Printf("Työhakemisto: %s", wd)

	iniFile := "massikone.ini"
	err := gotenv.Load(iniFile)
	if err == nil {
		log.Printf("Asetustiedosto %s käytössä", iniFile)
	} else if os.IsNotExist(err) {
		log.Printf("Asetustiedostoa %s ei ole työhakemistossa", iniFile)
	} else {
		log.Fatalf("Asetustiedoston %s käyttö epäonnistui. %s", iniFile, err)
	}
	if os.Getenv("DATABASE_URL") == "" {
		os.Setenv("DATABASE_URL", "sqlite://massikone.db")
	}
	model.Initialize()

	publicURL = os.Getenv("PUBLIC_URL")
	if publicURL != "" {
		cookieStore = sessions.NewCookieStore(
			getSessionSecret(os.Getenv("SESSION_SECRET")))
		gothic.Store = cookieStore
		goth.UseProviders(
			gplus.New(
				os.Getenv("GOOGLE_CLIENT_ID"),
				os.Getenv("GOOGLE_CLIENT_SECRET"),
				publicURL+"/auth/gplus/callback"),
		)
	}

	router := mux.NewRouter()

	get := func(path string, h http.HandlerFunc) {
		router.NewRoute().Path(path).Handler(h).Methods("GET")
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
	get(`/tosite/{billID}`,
		anyUser(getBillID))
	post(`/tosite/{billID}`,
		anyUser(putBillID))
	get(`/tosite`,
		anyUser(getNewBillPage))
	post(`/tosite`,
		anyUser(postBill))

	post(`/api/settings`,
		adminOnly(putSettings))
	//get(`/api/compare`,
	//	adminOnly(getCompare))
	get(`/asetukset`,
		adminOnly(getSettings))
	get(`/vertaa`,
		adminOnly(getCompare))
	get(`/raportti/tuloslaskelma`,
		adminOnly(report(reports.IncomeStatementPdf)))
	get(`/raportti/tuloslaskelma-erittelyin`,
		adminOnly(report(reports.IncomeStatementDetailedPdf)))
	get(`/raportti/tase`,
		adminOnly(report(reports.BalanceSheetPdf)))
	get(`/raportti/tase-erittelyin`,
		adminOnly(report(reports.BalanceSheetDetailedPdf)))
	get(`/raportti/paivakirja`,
		adminOnly(report(reports.GeneralJournalPdf)))
	get(`/raportti/paakirja`,
		adminOnly(report(reports.GeneralLedgerPdf)))
	get(`/raportti/tilikartta`,
		adminOnly(report(reports.ChartOfAccountsPdf)))
	get(`/raportti/tilinpaatos`,
		adminOnly(report(reports.FullStatementZip)))

	get(`/auth/{provider}/callback`, finishLogin)
	get(`/auth/{provider}`, gothic.BeginAuthHandler)
	get(`/ulos`, logout)
	post(`/ulos`, logout)
	get(`/`, getBillsOrLogin)
	router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/",
			http.FileServer(airfreight.HTTPFileSystem(static))))

	port := os.Getenv("PORT")
	addr := ""
	if strings.Contains(port, ":") {
		addr = port
	} else if publicURL == "" {
		if port == "" {
			port = "0"
		}
		addr = "127.0.0.1:" + port
	} else {
		if port == "" {
			log.Fatal("Julkiselle Massikoneelle täytyy määritellä PORT")
		}
		addr = ":" + port
	}
	listener, err := net.Listen("tcp", addr)
	check(err)
	privateURL := "http://" + listener.Addr().String()
	log.Printf("Yksityinen web-osoite: %s", privateURL)
	if publicURL != "" {
		log.Printf("Julkinen web-osoite: %s", publicURL)
		log.Print("Käytettävissä julkisesti, kirjautuminen vaadittu")
	} else {
		log.Print("Käytettävissä vain tällä tietokoneella")
		if err = webbrowser.Open(privateURL); err != nil {
			log.Printf("Selaimen avaaminen ei onnistunut. %s", err)
		}
	}
	check(http.Serve(listener,
		handlers.LoggingHandler(os.Stdout,
			handlers.RecoveryHandler(
				handlers.PrintRecoveryStack(true))(router))))
}
