package router

import (
	"database/sql"
	"leonlib/internal/handler"
	"net/http"

	"github.com/gorilla/mux"
)

type Router struct {
	Name        string
	Method      string
	Path        string
	HandlerFunc http.HandlerFunc
}

type Routes []Router

var routes Routes

func initRoutes(db *sql.DB) {
	routes = Routes{
		Router{
			"About Page",
			"GET",
			"/about",
			func(w http.ResponseWriter, r *http.Request) {
				handler.AboutPage(w, r)
			},
		},
		Router{
			"All Books",
			"GET",
			"/allbooks",
			func(w http.ResponseWriter, r *http.Request) {
				handler.AllBooksPage(db, w, r)
			},
		},
		Router{
			"Add Book Page",
			"GET",
			"/admin/add",
			func(w http.ResponseWriter, r *http.Request) {
				handler.AddBookPage(w, r)
			},
		},
		Router{
			"Add Book",
			"POST",
			"/addbook",
			func(w http.ResponseWriter, r *http.Request) {
				handler.AddBook(db, w, r)
			},
		},
		Router{
			"CheckLikeStatus",
			"GET",
			"/api/check_like/{word_id}",
			func(w http.ResponseWriter, r *http.Request) {
				handler.CheckLikeStatus(db, w, r)
			},
		},
		Router{
			"",
			"GET",
			"/admin/initdb",
			func(w http.ResponseWriter, r *http.Request) {
				handler.CreateDBFromFile(db, w)
			},
		},
		Router{
			"LikesCount",
			"GET",
			"/api/likes_count",
			func(w http.ResponseWriter, r *http.Request) {
				handler.LikesCount(db, w, r)
			},
		},
		Router{
			"Like Book",
			"POST",
			"/api/like",
			func(w http.ResponseWriter, r *http.Request) {
				handler.LikeBook(db, w, r)
			},
		},
		Router{
			"UnlikeWord",
			"DELETE",
			"/api/like",
			func(w http.ResponseWriter, r *http.Request) {
				handler.UnlikeBook(db, w, r)
			},
		},
		//Router{
		//	"GoogleAuth",
		//	"GET",
		//	"/auth/google/login",
		//	handler.GoogleLogin,
		//},
		//Router{
		//	"GoogleCallback",
		//	"GET",
		//	"/auth/callback",
		//	func(w http.ResponseWriter, r *http.Request) {
		//		handler.GoogleCallback(db, w, r)
		//	},
		//},
		Router{
			"Auth0Callback",
			"GET",
			"/auth/callback",
			func(w http.ResponseWriter, r *http.Request) {
				handler.Auth0Callback(db, w, r)
			},
		},
		Router{
			"Books by author",
			"GET",
			"/books_by_author",
			func(w http.ResponseWriter, r *http.Request) {
				handler.BooksByAuthorPage(db, w, r)
			},
		},
		Router{
			"Contact page",
			"GET",
			"/contact",
			func(w http.ResponseWriter, r *http.Request) {
				handler.ContactPage(w, r)
			},
		},
		Router{
			"ErrorPage",
			"GET",
			"/error",
			handler.ErrorPage,
		},
		Router{
			"IndexPage",
			"GET",
			"/",
			handler.IndexPage,
		},
		Router{
			"Search for books",
			"GET",
			"/search_books",
			func(w http.ResponseWriter, r *http.Request) {
				handler.SearchBooksPage(db, w, r)
			},
		},
		Router{
			"Book Info",
			"GET",
			"/book_info",
			func(w http.ResponseWriter, r *http.Request) {
				handler.InfoBook(db, w, r)
			},
		},
		Router{
			"Modify Book Page",
			"GET",
			"/admin/modify",
			func(w http.ResponseWriter, r *http.Request) {
				handler.ModifyBookPage(db, w, r)
			},
		},
		Router{
			"Modify Book",
			"POST",
			"/modify",
			func(w http.ResponseWriter, r *http.Request) {
				handler.ModifyBook(db, w, r)
			},
		},
		Router{
			"IngresarPage",
			"GET",
			"/ingresar",
			handler.IngresarPage,
		},
		//Router{
		//	"Autocomplete",
		//	"GET",
		//	"/api/autocomplete",
		//	func(w http.ResponseWriter, r *http.Request) {
		//		handler.Autocomplete(db, w, r)
		//	},
		//},
		Router{
			"Books Count",
			"GET",
			"/api/booksCount",
			func(w http.ResponseWriter, r *http.Request) {
				handler.BooksCount(db, w)
			},
		},
		Router{
			"Books List",
			"GET",
			"/api/books",
			func(w http.ResponseWriter, r *http.Request) {
				handler.BooksList(db, w, r)
			},
		},
		Router{
			"Remove Image",
			"POST",
			"/removeimage",
			func(w http.ResponseWriter, r *http.Request) {
				handler.RemoveImage(db, w, r)
			},
		},
	}
}

func NewRouter(db *sql.DB) *mux.Router {
	initRoutes(db)
	router := mux.NewRouter().StrictSlash(true)

	//rateLimiter := middleware.NewRateLimiterMiddleware(ratelimit.RedisClient, 1, 5)

	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Path).
			Name(route.Name).
			Handler(route.HandlerFunc)
	}

	fs := http.FileServer(http.Dir("assets/"))
	router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", fs))

	return router
}
