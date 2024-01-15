package router

import (
	"leonlib/internal/dao"
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

func initRoutes(dao *dao.DAO) {
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
				handler.AllBooksPage(dao, w, r)
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
				handler.AddBook(dao, w, r)
			},
		},
		Router{
			"CheckLikeStatus",
			"GET",
			"/api/check_like/{word_id}",
			func(w http.ResponseWriter, r *http.Request) {
				handler.CheckLikeStatus(dao, w, r)
			},
		},
		Router{
			"",
			"GET",
			"/admin/initdb",
			func(w http.ResponseWriter, r *http.Request) {
				handler.CreateDBFromFile(dao, w)
			},
		},
		Router{
			"LikesCount",
			"GET",
			"/api/likes_count",
			func(w http.ResponseWriter, r *http.Request) {
				handler.LikesCount(dao, w, r)
			},
		},
		Router{
			"Like Book",
			"POST",
			"/api/like",
			func(w http.ResponseWriter, r *http.Request) {
				handler.LikeBook(dao, w, r)
			},
		},
		Router{
			"UnlikeWord",
			"DELETE",
			"/api/like",
			func(w http.ResponseWriter, r *http.Request) {
				handler.UnlikeBook(dao, w, r)
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
				handler.Auth0Callback(dao, w, r)
			},
		},
		Router{
			"Books by author",
			"GET",
			"/books_by_author",
			func(w http.ResponseWriter, r *http.Request) {
				handler.BooksByAuthorPage(dao, w, r)
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
				handler.SearchBooksPage(dao, w, r)
			},
		},
		Router{
			"Book Info",
			"GET",
			"/book_info",
			func(w http.ResponseWriter, r *http.Request) {
				handler.InfoBook(dao, w, r)
			},
		},
		Router{
			"Modify Book Page",
			"GET",
			"/admin/modify",
			func(w http.ResponseWriter, r *http.Request) {
				handler.ModifyBookPage(dao, w, r)
			},
		},
		Router{
			"Modify Book",
			"POST",
			"/modify",
			func(w http.ResponseWriter, r *http.Request) {
				handler.ModifyBook(dao, w, r)
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
				handler.BooksCount(dao, w)
			},
		},
		Router{
			"Books List",
			"GET",
			"/api/books",
			func(w http.ResponseWriter, r *http.Request) {
				handler.BooksList(dao, w, r)
			},
		},
		Router{
			"Remove Image",
			"POST",
			"/removeimage",
			func(w http.ResponseWriter, r *http.Request) {
				handler.RemoveImage(dao, w, r)
			},
		},
	}
}

func NewRouter(dao *dao.DAO) *mux.Router {
	initRoutes(dao)
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
