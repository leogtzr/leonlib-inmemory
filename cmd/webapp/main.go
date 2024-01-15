package main

import (
	"context"
	"database/sql"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
	"leonlib/internal/auth"
	"leonlib/internal/captcha"
	"leonlib/internal/dao"
	"leonlib/internal/router"
	"log"
	"net/http"
	"os"
)

var (
	dbMode      = os.Getenv("DB_MODE")
	dbHost      = os.Getenv("PGHOST")
	dbUser      = os.Getenv("PGUSER")
	dbPassword  = os.Getenv("POSTGRES_PASSWORD")
	dbName      = os.Getenv("PGDATABASE")
	dbPort      = os.Getenv("PGPORT")
	ctx         = context.Background()
	mainAppUser = os.Getenv("LEONLIB_MAINAPP_USER")
	DB          *sql.DB
)

func init() {
	if dbMode == "" {
		dbMode = "memory"
		// log.Fatal("error: DB_MODE not defined")
	}
	if mainAppUser == "" {
		log.Fatal("error: LEONLIB_MAINAPP_USER not defined")
	}
	captcha.SiteKey = os.Getenv("LEONLIB_CAPTCHA_SITE_KEY")
	captcha.SecretKey = os.Getenv("LEONLIB_CAPTCHA_SECRET_KEY")
	if captcha.SiteKey == "" {
		log.Fatal("error: LEONLIB_CAPTCHA_SITE_KEY not defined")
	}
	if captcha.SecretKey == "" {
		log.Fatal("error: LEONLIB_CAPTCHA_SECRET_KEY not defined")
	}

	auth.Config = &oauth2.Config{
		ClientID:     os.Getenv("AUTH0_CLIENT_ID"),
		ClientSecret: os.Getenv("AUTH0_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("AUTH0_CALLBACK_URL"),
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://" + os.Getenv("AUTH0_DOMAIN") + "/authorize",
			TokenURL: "https://" + os.Getenv("AUTH0_DOMAIN") + "/oauth/token",
		},
	}

	auth.SessionStore = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))
}

func main() {
	dao, err := dao.NewDAO(dbMode, dbHost, dbPort, dbUser, dbPassword, dbName)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = dao.Close()
	}()

	err = dao.Ping()
	if err != nil {
		panic(err)
	}

	r := router.NewRouter(&dao)

	fs := http.FileServer(http.Dir("assets/"))
	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", fs))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8180"
	}

	log.Printf("Listening on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
