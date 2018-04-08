package main

import (
	"fmt"
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

func setCurrentUser(w http.ResponseWriter, r *http.Request, id string) {
	session, _ := store.Get(r, sessionName)
	if id == "" {
		delete(session.Values, sessionCurrentUser)
	} else {
		session.Values[sessionCurrentUser] = id
	}
	session.Save(r, w)
}

func getCurrentUser(r *http.Request) string {
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
	setCurrentUser(w, r, user.Name)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func postLogout(w http.ResponseWriter, r *http.Request) {
	setCurrentUser(w, r, "")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getBillsOrLogin(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == "" {
		w.Write([]byte(loginTemplate.Render(
			map[string]string{"app_title": getAppTitle()})))
		return
	}
	bills := ModelGetBills()
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

func getBill(w http.ResponseWriter, r *http.Request) {
	//user := getCurrentUser(r)
	w.Write([]byte(billTemplate.Render(
		map[string]string{"app_title": getAppTitle()})))
}

func getCompare(w http.ResponseWriter, r *http.Request) {
	//user := getCurrentUser(r)
	w.Write([]byte(compareTemplate.Render(
		map[string]string{"app_title": getAppTitle()})))
}

func getUserImage(w http.ResponseWriter, r *http.Request) {
	// TODO: http header, esp. caching
	imageId := r.URL.Query().Get(":imageId")
	imageData, err := ModelGetImageData(imageId)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg") //TODO
	w.Write(imageData)
}

func main() {
	p := pat.New()
	p.Get(`/api/userimage/{imageId:[0-9a-f]{40}\.(?:jpeg|png)}`, getUserImage)
	p.Get("/auth/{provider}/callback", authCallbackHandler)
	p.Get("/auth/{provider}", gothic.BeginAuthHandler)
	p.Post("/logout", postLogout)
	p.Get("/compare", getCompare)
	p.Get("/bill", getBill)
	p.Get("/", getBillsOrLogin)

	mux := http.NewServeMux()
	mux.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(staticBox)))
	mux.Handle("/", p)
	log.Print("Starting web server")
	http.ListenAndServe(":"+port, mux)
}
