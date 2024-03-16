package router

import (
	"golang.org/x/time/rate"
	"leonlib/internal/dao"
	"leonlib/internal/handler"
	"leonlib/internal/middleware"
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
			Name:   "About Page",
			Method: "GET",
			Path:   "/about",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.AboutPage(w, r)
			},
		},
		Router{
			Name:   "All Books",
			Method: "GET",
			Path:   "/allbooks",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.AllBooksPage(dao, w, r)
			},
		},
		Router{
			Name:   "Add Book Page",
			Method: "GET",
			Path:   "/admin/add",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.AddBookPage(dao, w, r)
			},
		},
		Router{
			Name:   "Add Book",
			Method: "POST",
			Path:   "/addbook",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.AddBook(dao, w, r)
			},
		},
		Router{
			Name:   "Check Like Status",
			Method: "GET",
			Path:   "/api/check_like/{book_id}",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.CheckLikeStatus(dao, w, r)
			},
		},
		Router{
			Name:   "Init Database",
			Method: "GET",
			Path:   "/admin/initdb",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.CreateDBFromFile(dao, w)
			},
		},
		Router{
			Name:   "Likes Count",
			Method: "GET",
			Path:   "/api/likes_count",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.LikesCount(dao, w, r)
			},
		},
		Router{
			Name:   "Like Book",
			Method: "POST",
			Path:   "/api/like",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.LikeBook(dao, w, r)
			},
		},
		Router{
			Name:   "Unlike Book",
			Method: "DELETE",
			Path:   "/api/like",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
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
			Name:   "Auth0Callback",
			Method: "GET",
			Path:   "/auth/callback",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.Auth0Callback(dao, w, r)
			},
		},
		Router{
			Name:   "Books by author",
			Method: "GET",
			Path:   "/books_by_author",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.BooksByAuthorPage(dao, w, r)
			},
		},
		Router{
			Name:   "Contact page",
			Method: "GET",
			Path:   "/contact",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.ContactPage(w, r)
			},
		},
		Router{
			Name:        "ErrorPage",
			Method:      "GET",
			Path:        "/error",
			HandlerFunc: handler.ErrorPage,
		},
		Router{
			Name:        "IndexPage",
			Method:      "GET",
			Path:        "/",
			HandlerFunc: handler.IndexPage,
		},
		Router{
			Name:   "Search for books",
			Method: "GET",
			Path:   "/search_books",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.SearchBooksPage(dao, w, r)
			},
		},
		Router{
			Name:   "Book Info",
			Method: "GET",
			Path:   "/book_info",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.InfoBook(dao, w, r)
			},
		},
		Router{
			Name:   "Modify Book Page",
			Method: "GET",
			Path:   "/admin/modify",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.ModifyBookPage(dao, w, r)
			},
		},
		Router{
			Name:   "Modify Book",
			Method: "POST",
			Path:   "/modify",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.ModifyBook(dao, w, r)
			},
		},
		Router{
			Name:        "IngresarPage",
			Method:      "GET",
			Path:        "/ingresar",
			HandlerFunc: handler.IngresarPage,
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
			Name:   "Books Count",
			Method: "GET",
			Path:   "/api/booksCount",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.BooksCount(dao, w)
			},
		},
		Router{
			Name:   "Books List",
			Method: "GET",
			Path:   "/api/books",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.BooksList(dao, w, r)
			},
		},
		Router{
			Name:   "Remove Image",
			Method: "POST",
			Path:   "/removeimage",
			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
				handler.RemoveImage(dao, w, r)
			},
		},
	}
}

func NewRouter(dao *dao.DAO, limiter *rate.Limiter) *mux.Router {
	initRoutes(dao)
	router := mux.NewRouter().StrictSlash(true)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Tu lógica aquí
	})

	rateLimitMiddleware := middleware.RateLimitMiddlewareAdapter(limiter, nextHandler)
	router.Use(rateLimitMiddleware)

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
