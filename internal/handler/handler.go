package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"html/template"
	"io"
	"leonlib/internal/auth"
	"leonlib/internal/captcha"
	"leonlib/internal/dao"
	book "leonlib/internal/types"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/BurntSushi/toml"
)

const numberOfResultsByPage = 20

type RequestData struct {
	BookID string `json:"book_id"`
}

type PageVariables struct {
	Year     string
	SiteKey  string
	LoggedIn bool
}

type PageVariablesForAuthors struct {
	Year     string
	SiteKey  string
	Authors  []string
	LoggedIn bool
}

type PageResultsVariables struct {
	Year         string
	SiteKey      string
	Results      []book.BookInfo
	LoggedIn     bool
	Funcs        template.FuncMap
	Page         int
	TotalPages   int
	CurrentPage  int
	PreviousPage int
	NextPage     int
	StartPage    int
	EndPage      int
	Pages        []int
}

type UserInfo struct {
	Sub      string `json:"sub"`            // Identificador único del usuario
	Name     string `json:"name"`           // Nombre completo del usuario
	Nickname string `json:"nickname"`       // Apodo del usuario
	Picture  string `json:"picture"`        // URL de la imagen de perfil del usuario
	Email    string `json:"email"`          // Correo electrónico del usuario
	Verified bool   `json:"email_verified"` // Si el correo electrónico está verificado
}

// LikeStatus { "status" : "error" | "liked" | "not-liked" }
type LikeStatus struct {
	Status string
}

func (ui UserInfo) String() string {
	return fmt.Sprintf("Name=(%s), email=(%s), nickname=(%s), verified=(%t), sub=(%s)", ui.Name, ui.Email, ui.Nickname, ui.Verified, ui.Sub)
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}

	return base64.URLEncoding.EncodeToString(b)
}

func getDatabaseEmailFromSessionID(db *sql.DB, userID string) (string, error) {
	queryStr := "SELECT u.email FROM users u WHERE u.user_id=$1"

	rows, err := db.Query(queryStr, userID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var email string

	if rows.Next() {
		if err := rows.Scan(&email); err != nil {
			return "", err
		}
	}

	return email, nil
}

func parseBookSearchType(input string) book.BookSearchType {
	switch strings.TrimSpace(strings.ToLower(input)) {
	case "bytitle":
		return book.ByTitle
	case "byauthor":
		return book.ByAuthor
	default:
		return book.Unknown
	}
}

func getDevUserInfo() (*UserInfo, error) {
	if !isDevMode() {
		return nil, fmt.Errorf("no dev mode")
	}

	var userInfo UserInfo
	userInfo.Nickname = "Leo"
	userInfo.Name = "Leo"
	userInfo.Email = os.Getenv("LEONLIB_MAINAPP_USER")
	userInfo.Sub = "leonardo"
	userInfo.Verified = true

	return &userInfo, nil
}

func getUserInfoFromAuth0(accessToken string) (*UserInfo, error) {
	userInfoEndpoint := fmt.Sprintf("https://%s/userinfo", os.Getenv("AUTH0_DOMAIN"))

	req, err := http.NewRequest("GET", userInfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creando la solicitud: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error al realizar la solicitud: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error al leer la respuesta: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error en la respuesta de Auth0: %s", body)
	}

	var userInfo UserInfo
	err = json.Unmarshal(body, &userInfo)
	if err != nil {
		return nil, fmt.Errorf("error al decodificar la respuesta JSON: %v", err)
	}

	return &userInfo, nil
}

func redirectToErrorPage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/error", http.StatusSeeOther)
}

func getTemplatePath(templateFileName string) string {
	templateDir := os.Getenv("TEMPLATE_DIR")
	if templateDir == "" {
		templateDir = "internal/template" // default value for local development
	}
	templatePath := filepath.Join(templateDir, templateFileName)

	return templatePath
}

func redirectToErrorPageWithMessageAndStatusCode(w http.ResponseWriter, errorMessage string, httpStatusCode int) {
	templatePath := getTemplatePath("error5xx.html")

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Printf("error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type ErrorVariables struct {
		Year         string
		ErrorMessage string
	}

	now := time.Now()

	pageVariables := ErrorVariables{
		Year:         now.Format("2006"),
		ErrorMessage: errorMessage,
	}

	w.WriteHeader(httpStatusCode)

	err = t.Execute(w, pageVariables)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}
}

func writeErrorGeneralStatus(w http.ResponseWriter, err error) {
	log.Printf("error: %v", err)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "error",
	})
}

func writeErrorLikeStatus(w http.ResponseWriter, err error) {
	log.Printf("Error parsing template: %v", err)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "error",
	})
}

func writeUnauthenticated(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(map[string]string{"status": "unauthenticated"})
}

func getCurrentUserID(r *http.Request) (string, error) {
	session, err := auth.SessionStore.Get(r, "user-session")
	if err != nil {
		return "", err
	}

	userID, ok := session.Values["user_id"].(string)
	if !ok {
		return "", errors.New("0) user_id not found in session")
	}

	fmt.Println("--------")
	fmt.Println(session)
	fmt.Println(userID)
	fmt.Println("----- end")

	return userID, nil
}

func getAllAuthors(db *sql.DB) ([]string, error) {
	var err error

	allAuthorsRows, err := db.Query("SELECT DISTINCT author FROM books ORDER BY author")
	if err != nil {
		return []string{}, err
	}

	defer allAuthorsRows.Close()

	var authors []string
	for allAuthorsRows.Next() {
		var author string
		if err := allAuthorsRows.Scan(&author); err != nil {
			return []string{}, err
		}
		authors = append(authors, author)
	}

	return authors, nil
}

func uniqueSearchTypes(searchTypes []string) []string {
	set := make(map[string]struct{})
	var result []string

	for _, item := range searchTypes {
		if _, exists := set[item]; !exists {
			set[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

func IndexPage(w http.ResponseWriter, r *http.Request) {
	now := time.Now()

	pageVariables := PageVariables{
		Year:    now.Format("2006"),
		SiteKey: captcha.SiteKey,
	}

	_, err := getCurrentUserID(r)
	if err != nil {
		log.Printf("User is not logged in: %v", err)
		pageVariables.LoggedIn = false
	} else {
		log.Println("User is logged in")
		pageVariables.LoggedIn = true
	}

	templatePath := getTemplatePath("index.html")

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		redirectToErrorPageWithMessageAndStatusCode(w, err.Error(), http.StatusInternalServerError)

		return
	}

	err = t.Execute(w, pageVariables)
	if err != nil {
		log.Printf("Error executing template: %v", err)
		redirectToErrorPageWithMessageAndStatusCode(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

func BooksByAuthorPage(dao *dao.DAO, w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	pageVariables := PageVariablesForAuthors{
		Year:    now.Format("2006"),
		SiteKey: captcha.SiteKey,
	}

	authors, err := (*dao).GetAllAuthors()
	if err != nil {
		log.Printf("Error getting authors: %v", err)
		redirectToErrorPageWithMessageAndStatusCode(w, "error getting information from the database", http.StatusInternalServerError)
		return
	}

	_, err = getCurrentUserID(r)
	if err != nil {
		log.Printf("(BooksByAuthorPage) User is not logged in: %v", err)
		pageVariables.LoggedIn = false
	} else {
		log.Println("User is logged in")
		pageVariables.LoggedIn = true
	}

	pageVariables.Authors = authors

	templatePath := getTemplatePath("books_by_author.html")

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error parsing template: %v", err)
		return
	}

	err = t.Execute(w, pageVariables)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error executing template: %v", err)
	}
}

func getBooksWithPagination(db *sql.DB, offset, limit int) ([]book.BookInfo, error) {
	query := `SELECT id, title, author, description, read, added_on FROM books ORDER BY title LIMIT $1 OFFSET $2;`

	fmt.Printf("query=(%s)\n", query)

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	books := []book.BookInfo{}
	for rows.Next() {
		book := book.BookInfo{}
		err = rows.Scan(&book.ID, &book.Title, &book.Author, &book.Description, &book.HasBeenRead, &book.AddedOn)
		if err != nil {
			return nil, err
		}
		books = append(books, book)
	}

	return books, nil
}

func setUpPaginationFor(pageInt int, dao *dao.DAO, pageVariables *PageResultsVariables) error {
	now := time.Now()

	pageVariables.Year = now.Format("2006")
	pageVariables.SiteKey = captcha.SiteKey

	totalBooks, err := (*dao).GetBookCount()
	if err != nil {
		log.Printf("Error getting total books: %v", err)
		return err
	}

	totalPages := int(math.Ceil(float64(totalBooks) / float64(numberOfResultsByPage)))
	pageVariables.TotalPages = totalPages
	pageVariables.PreviousPage = pageInt - 1
	pageVariables.CurrentPage = pageInt
	pageVariables.NextPage = pageInt + 1
	pageVariables.LoggedIn = false
	pageVariables.StartPage = 1
	pageVariables.EndPage = totalPages

	start := 1
	end := totalPages

	if totalPages > 5 {
		if pageInt > 3 {
			start = pageInt - 2
			end = pageInt + 2
			if end > totalPages {
				end = totalPages
				start = end - 4
			}
		} else {
			end = 5
		}
	}

	var pages []int
	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}

	pageVariables.Pages = pages

	pageVariables.StartPage = start
	pageVariables.EndPage = end

	return nil
}

func AllBooksPage(dao *dao.DAO, w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	if page == "" {
		page = "1"
	}

	pageInt, err := strconv.Atoi(page)
	if err != nil {
		log.Printf("Error converting page to int: %v", err)
		redirectToErrorPageWithMessageAndStatusCode(w, "error getting information from the database", http.StatusInternalServerError)
		return
	}

	offset := (pageInt - 1) * numberOfResultsByPage

	books, err := (*dao).GetBooksWithPagination(offset, numberOfResultsByPage)
	if err != nil {
		log.Printf("Error getting books: %v", err)
		redirectToErrorPageWithMessageAndStatusCode(w, "error getting information from the database", http.StatusInternalServerError)
		return
	}

	pageVariables := PageResultsVariables{}
	pageVariables.Results = books

	err = setUpPaginationFor(pageInt, dao, &pageVariables)
	if err != nil {
		log.Printf("Error setting up pagination: %v", err)
		redirectToErrorPageWithMessageAndStatusCode(w, "error getting information from the database", http.StatusInternalServerError)
		return
	}

	setAuthenticationForPageResults(r, &pageVariables)

	templateDir := os.Getenv("TEMPLATE_DIR")
	if templateDir == "" {
		templateDir = "internal/template" // default value for local development
	}
	templatePath := getTemplatePath("allbooks.html")

	t := template.New("").Funcs(sprig.TxtFuncMap())

	tmpl, err := t.ParseFiles(templatePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error parsing template: %v", err)
		return
	}

	err = tmpl.ExecuteTemplate(w, "allbooks.html", pageVariables)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error executing template: %v", err)
	}
}

//func Autocomplete(db *sql.DB, w http.ResponseWriter, r *http.Request) {
//	query := r.URL.Query().Get("q")
//
//	searchTypesStr := r.URL.Query().Get("searchType")
//	searchTypes := strings.Split(searchTypesStr, ",")
//
//	var suggestions []string
//
//	var queryStr string
//	var rows *sql.Rows
//	var err error
//
//	// Perform DB query based on queryParam("q")
//
//	w.Header().Set("Content-Type", "application/json")
//	json.NewEncoder(w).Encode(map[string][]string{
//		"suggestions": suggestions,
//	})
//}

func BooksList(dao *dao.DAO, w http.ResponseWriter, r *http.Request) {
	authorParam := r.URL.Query().Get("start_with")

	booksByAuthor, err := (*dao).GetBooksBySearchTypeCoincidence(authorParam, book.ByAuthor)
	if err != nil {
		log.Printf("error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	type BookDetail struct {
		ID           int                  `json:"id"`
		Title        string               `json:"title"`
		Author       string               `json:"author"`
		Description  string               `json:"description"`
		Base64Images []book.BookImageInfo `json:"images"`
	}

	var results []BookDetail

	for _, book := range booksByAuthor {
		bookDetail := BookDetail{}
		bookDetail.ID = book.ID
		bookDetail.Title = book.Title
		bookDetail.Author = book.Author
		bookDetail.Description = book.Description
		bookDetail.Base64Images = book.Base64Images

		results = append(results, bookDetail)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}

func getTotalBooks(db *sql.DB) (int, error) {
	queryStr := `SELECT count(*) FROM books`
	rows, err := db.Query(queryStr)
	if err != nil {
		return -1, err
	}

	var count int

	for rows.Next() {
		err := rows.Scan(&count)
		if err != nil {
			return -1, err
		}
	}

	return count, nil
}

func BooksCount(dao *dao.DAO, w http.ResponseWriter) {
	count, err := (*dao).GetBookCount()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{
		"booksCount": count,
	})
}

func SearchBooksPage(dao *dao.DAO, w http.ResponseWriter, r *http.Request) {
	bookQuery := r.URL.Query().Get("textSearch")
	searchTypesStr := r.URL.Query().Get("searchType")
	searchTypesParams := uniqueSearchTypes(strings.Split(searchTypesStr, ","))

	if len(searchTypesParams) == 0 || (len(searchTypesParams) == 1 && searchTypesParams[0] == "") {
		searchTypesParams = []string{"byTitle"}
	}

	var results []book.BookInfo
	var err error

	for _, searchTypeParam := range searchTypesParams {
		searchType := parseBookSearchType(searchTypeParam)
		switch searchType {
		case book.ByTitle:
			booksByTitle, err := (*dao).GetBooksBySearchTypeCoincidence(bookQuery, book.ByTitle)
			if err != nil {
				log.Printf("error: %v", err)
				redirectToErrorPageWithMessageAndStatusCode(w, "Error getting information from the database", http.StatusInternalServerError)

				return
			}
			results = append(results, booksByTitle...)

		case book.ByAuthor:
			booksByAuthor, err := (*dao).GetBooksBySearchTypeCoincidence(bookQuery, book.ByAuthor)
			if err != nil {
				log.Printf("error getting info from the database: %v", err)
				redirectToErrorPageWithMessageAndStatusCode(w, "error getting info from the database", http.StatusInternalServerError)
				return
			}
			results = append(results, booksByAuthor...)

		case book.Unknown:
			log.Printf("Tipo de búsqueda en libros desconocido.")
			redirectToErrorPageWithMessageAndStatusCode(w, "Wrong search", http.StatusInternalServerError)

			return
		}
	}

	now := time.Now()
	pageVariables := PageResultsVariables{
		Year:    now.Format("2006"),
		SiteKey: captcha.SiteKey,
		Results: results,
	}

	templatePath := getTemplatePath("search_books.html")

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Printf("template error: %v", err)
		redirectToErrorPageWithMessageAndStatusCode(w, "template error", http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, pageVariables)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("error: %v", err)
		return
	}
}

func ErrorPage(w http.ResponseWriter, _ *http.Request) {
	templateDir := os.Getenv("TEMPLATE_DIR")
	if templateDir == "" {
		templateDir = "internal/template"
	}
	templatePath := getTemplatePath("error5xx.html")

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func IngresarPage(w http.ResponseWriter, r *http.Request) {
	oauthState := generateRandomString(32)

	session, _ := auth.SessionStore.Get(r, "user-session")
	session.Values["oauth_state"] = oauthState
	session.Save(r, w)

	if isDevMode() {
		http.Redirect(w, r, "/auth/callback", http.StatusSeeOther)

		return
	}

	//url := auth.GoogleOauthConfig.AuthCodeURL(oauthState)
	url := auth.Config.AuthCodeURL(oauthState)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func setDevCredentialsAndRedirect(dao *dao.DAO, w http.ResponseWriter, r *http.Request) {
	userInfo, err := getDevUserInfo()
	if err != nil {
		log.Printf("error: cannot get user info from Auth0: %v", err)
		http.Error(w, "cannot get user info from Auth0", http.StatusInternalServerError)

		return
	}

	err = (*dao).AddUser(userInfo.Sub, userInfo.Email, userInfo.Name, "Google")
	if err != nil {
		http.Error(w, "Error al guardar el usuario en la base de datos", http.StatusInternalServerError)
		return
	}

	session, _ := auth.SessionStore.Get(r, "user-session")
	session.Values["user_id"] = userInfo.Sub
	session.Save(r, w)

	now := time.Now()

	pageVariables := PageVariables{
		Year:    now.Format("2006"),
		SiteKey: captcha.SiteKey,
	}

	_, err = getCurrentUserID(r)
	if err != nil {
		pageVariables.LoggedIn = false
	} else {
		pageVariables.LoggedIn = true
	}

	templateDir := os.Getenv("TEMPLATE_DIR")
	if templateDir == "" {
		templateDir = "internal/template"
	}
	templatePath := filepath.Join(templateDir, "index.html")

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error al analizar la plantilla: %v", err)
		return
	}

	err = t.Execute(w, pageVariables)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error al ejecutar la plantilla: %v", err)
	}
}

func Auth0Callback(dao *dao.DAO, w http.ResponseWriter, r *http.Request) {
	if isDevMode() {
		setDevCredentialsAndRedirect(dao, w, r)

		return
	}

	code := r.URL.Query().Get("code")

	token, err := auth.Config.Exchange(r.Context(), code)
	if err != nil {
		log.Printf("Error: %v", err)
		http.Error(w, "Cannot get Auth0 token", http.StatusInternalServerError)
		return
	}

	userInfo, err := getUserInfoFromAuth0(token.AccessToken)
	if err != nil {
		log.Printf("error: cannot get user info from Auth0: %v", err)
		http.Error(w, "cannot get user info from Auth0", http.StatusInternalServerError)
		return
	}

	err = (*dao).AddUser(userInfo.Sub, userInfo.Email, userInfo.Name, "Google")
	if err != nil {
		http.Error(w, "Error al guardar el usuario en la base de datos", http.StatusInternalServerError)
		return
	}

	session, _ := auth.SessionStore.Get(r, "user-session")
	session.Values["user_id"] = userInfo.Sub
	session.Save(r, w)

	now := time.Now()

	pageVariables := PageVariables{
		Year:    now.Format("2006"),
		SiteKey: captcha.SiteKey,
	}

	_, err = getCurrentUserID(r)
	if err != nil {
		pageVariables.LoggedIn = false
	} else {
		pageVariables.LoggedIn = true
	}

	templateDir := os.Getenv("TEMPLATE_DIR")
	if templateDir == "" {
		templateDir = "internal/template"
	}
	templatePath := filepath.Join(templateDir, "index.html")

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error al analizar la plantilla: %v", err)
		return
	}

	err = t.Execute(w, pageVariables)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error al ejecutar la plantilla: %v", err)
	}
}

func setAuthenticationForPageResults(r *http.Request, pageResultsVariables *PageResultsVariables) {
	if _, err := getCurrentUserID(r); err != nil {
		log.Printf("error: checking authentication information for user '%v'", err)
		pageResultsVariables.LoggedIn = false
	} else {
		pageResultsVariables.LoggedIn = true
	}
}

func CheckLikeStatus(dao *dao.DAO, w http.ResponseWriter, r *http.Request) {
	userID, err := getCurrentUserID(r)
	if err != nil {
		writeUnauthenticated(w)

		return
	}

	vars := mux.Vars(r)
	bookID := vars["book_id"]

	exists, err := (*dao).LikedBy(bookID, userID)
	if err != nil {
		writeErrorLikeStatus(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if exists {
		json.NewEncoder(w).Encode(map[string]string{"status": "liked"})
	} else {
		json.NewEncoder(w).Encode(map[string]string{"status": "not-liked"})
	}
}

func LikeBook(dao *dao.DAO, w http.ResponseWriter, r *http.Request) {
	userID, err := getCurrentUserID(r)
	if err != nil {
		http.Error(w, "Error al obtener información de la sesión", http.StatusInternalServerError)
		return
	}

	err = r.ParseForm()
	if err != nil {
		w.Write([]byte(fmt.Sprintf("error like book: %v", err.Error())))
	}
	bookID := r.PostFormValue("book_id")

	err = (*dao).LikeBook(bookID, userID)

	if err != nil {
		http.Error(w, "Error al dar like en la base de datos", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Liked successfully"))
}

func AddBook(dao *dao.DAO, w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(2 << 20)
	if err != nil {
		log.Printf("error adding book: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	book := book.BookInfo{}
	book.Title = r.FormValue("title")
	book.Author = r.FormValue("author")
	book.Description = r.FormValue("description")
	book.HasBeenRead = r.FormValue("read") == "on"
	book.GoodreadsLink = r.FormValue("goodreadsLink")

	var imageData []byte
	file, _, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		imageData, err = io.ReadAll(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if !errors.Is(err, http.ErrMissingFile) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	book.Image = imageData

	err = (*dao).CreateBook(book)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Libro agregado con éxito"))
}

func UnlikeBook(dao *dao.DAO, w http.ResponseWriter, r *http.Request) {
	userID, err := getCurrentUserID(r)
	if err != nil {
		http.Error(w, "Error al obtener información de la sesión", http.StatusInternalServerError)
		return
	}

	var requestData RequestData

	err = json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Error al decodificar el cuerpo de la solicitud", http.StatusInternalServerError)
		return
	}

	bookID := requestData.BookID

	fmt.Printf("debug:x trying to unlike book_id=(%s), user_id=(%s)\n", bookID, userID)

	err = (*dao).UnlikeBook(bookID, userID)
	if err != nil {
		http.Error(w, "Error al quitar el like en la base de datos", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Unliked successfully"))
}

func LikesCount(dao *dao.DAO, w http.ResponseWriter, r *http.Request) {
	bookID := r.URL.Query().Get("book_id")
	if bookID == "" {
		http.Error(w, "book_id is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(bookID)
	if err != nil {
		http.Error(w, "Invalid book_id", http.StatusBadRequest)
		return
	}

	var count int
	count, err = (*dao).LikesCount(id)
	if err != nil {
		http.Error(w, "Error querying the database", http.StatusInternalServerError)
		return
	}

	resp := map[string]int{
		"count": count,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func CreateDBFromFile(dao *dao.DAO, w http.ResponseWriter) {
	libraryDir := "library"
	libraryDirPath := filepath.Join(libraryDir, "books_db.toml")

	var library book.Library

	if _, err := toml.DecodeFile(libraryDirPath, &library); err != nil {
		writeErrorGeneralStatus(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	startTime := time.Now()

	err := (*dao).AddAll(library.Book)
	if err != nil {
		writeErrorGeneralStatus(w, err)
		return
	}

	elapsedTime := time.Since(startTime)

	log.Printf("Books loaded in: %.2f seconds\n", elapsedTime.Seconds())

	json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
}

func InfoBook(dao *dao.DAO, w http.ResponseWriter, r *http.Request) {
	idQueryParam := r.URL.Query().Get("id")

	id, err := strconv.Atoi(idQueryParam)
	if err != nil {
		redirectToErrorPage(w, r)
		return
	}

	bookByID, err := (*dao).GetBookByID(id)
	if err != nil {
		log.Printf("error: getting information from the database")
		redirectToErrorPageWithMessageAndStatusCode(w, "error getting information from the database", http.StatusInternalServerError)
		return
	}

	now := time.Now()

	pageVariables := PageResultsVariables{
		Year:    now.Format("2006"),
		SiteKey: captcha.SiteKey,
		Results: []book.BookInfo{bookByID},
	}

	setAuthenticationForPageResults(r, &pageVariables)

	templatePath := getTemplatePath("book_info.html")

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		redirectToErrorPage(w, r)
		return
	}

	err = t.Execute(w, pageVariables)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("template error: %v", err)
		return
	}
}

func ModifyBook(dao *dao.DAO, w http.ResponseWriter, r *http.Request) {
	// TODO: check auth here
	err := r.ParseMultipartForm(2 << 20)
	if err != nil {
		writeErrorGeneralStatus(w, err)

		return
	}

	bookIDParam := r.FormValue("book_id")
	title := r.FormValue("title")
	author := r.FormValue("author")
	description := r.FormValue("description")
	read := r.FormValue("read") == "on"
	goodreadsLink := r.FormValue("goodreadsLink")

	id, err := strconv.Atoi(bookIDParam)
	if err != nil {
		writeErrorGeneralStatus(w, err)

		return
	}

	err = addImageToBook(dao, id, r)
	if err != nil {
		writeErrorGeneralStatus(w, err)

		return
	}

	err = (*dao).UpdateBook(title, author, description, read, goodreadsLink, id)
	if err != nil {
		writeErrorGeneralStatus(w, err)

		return
	}

	w.Write([]byte("Libro modificado con exito"))
}

func addImageToBook(dao *dao.DAO, id int, r *http.Request) error {
	var imageData []byte
	file, _, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		imageData, err = io.ReadAll(file)
		if err != nil {
			return err
		}
	} else if !errors.Is(err, http.ErrMissingFile) {
		return err
	}

	if len(imageData) == 0 {
		return nil
	}

	err = (*dao).AddImageToBook(id, imageData)
	if err != nil {
		return err
	}

	return nil
}

func ModifyBookPage(dao *dao.DAO, w http.ResponseWriter, r *http.Request) {
	idQueryParam := r.URL.Query().Get("book_id")

	id, err := strconv.Atoi(idQueryParam)
	if err != nil {
		redirectToErrorPageWithMessageAndStatusCode(w, "wrong ID", http.StatusInternalServerError)
		return
	}

	bookByID, err := (*dao).GetBookByID(id)
	if err != nil {
		redirectToErrorPageWithMessageAndStatusCode(w, "error getting information from the database", http.StatusInternalServerError)
		return
	}

	now := time.Now()

	type BookToModifyVariables struct {
		Year          string
		SiteKey       string
		Book          book.BookInfo
		LoggedIn      bool
		GoodreadsLink template.URL
	}

	pageVariables := BookToModifyVariables{
		Year:          now.Format("2006"),
		SiteKey:       captcha.SiteKey,
		Book:          bookByID,
		GoodreadsLink: template.URL(bookByID.GoodreadsLink),
	}

	//_, err = getCurrentUserID(r)
	//if err != nil {
	//	redirectToErrorPageWithMessageAndStatusCode(w, "Error al obtener información de la sesión", http.StatusInternalServerError)
	//
	//	return
	//}
	pageVariables.LoggedIn = true

	templatePath := getTemplatePath("modify.html")

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		redirectToErrorPageWithMessageAndStatusCode(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, pageVariables)
	if err != nil {
		redirectToErrorPageWithMessageAndStatusCode(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

func AboutPage(w http.ResponseWriter, r *http.Request) {
	templatePath := getTemplatePath("about.html")

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		redirectToErrorPageWithMessageAndStatusCode(w, err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now()

	pageVariables := PageVariables{
		Year:     now.Format("2006"),
		SiteKey:  captcha.SiteKey,
		LoggedIn: false,
	}

	err = t.Execute(w, pageVariables)
	if err != nil {
		redirectToErrorPageWithMessageAndStatusCode(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ContactPage(w http.ResponseWriter, _ *http.Request) {
	templatePath := getTemplatePath("contact.html")

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		redirectToErrorPageWithMessageAndStatusCode(w, err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now()

	pageVariables := PageVariables{
		Year:     now.Format("2006"),
		SiteKey:  captcha.SiteKey,
		LoggedIn: false,
	}

	err = t.Execute(w, pageVariables)
	if err != nil {
		redirectToErrorPageWithMessageAndStatusCode(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func AddBookPage(w http.ResponseWriter, r *http.Request) {
	/*
		userID, err := getCurrentUserID(r)
		if err != nil {
			redirectToErrorPageWithMessageAndStatusCode(w, "Error al obtener información de la sesión", http.StatusInternalServerError)

			return
		}

		email, err := getDatabaseEmailFromSessionID(db, userID)

		if err != nil {
			redirectToErrorPageWithMessageAndStatusCode(w, "Only admins can access to this page", http.StatusForbidden)

			return
		}

		fmt.Printf("debug:x email=(%s)\n", email)
	*/

	templatePath := getTemplatePath("add_book.html")

	t, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Printf("template error: %v", err)
		redirectToErrorPageWithMessageAndStatusCode(w, fmt.Sprintf("template error: %v", err), http.StatusInternalServerError)
		return
	}

	now := time.Now()

	pageVariables := PageVariables{
		Year:     now.Format("2006"),
		SiteKey:  captcha.SiteKey,
		LoggedIn: false,
	}

	err = t.Execute(w, pageVariables)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("template error: %v", err)
		return
	}
}

func RemoveImage(dao *dao.DAO, w http.ResponseWriter, r *http.Request) {
	// TODO: check auth
	r.ParseForm()
	imageIDParam := r.PostFormValue("image_id")

	imageID, err := strconv.Atoi(imageIDParam)
	if err != nil {
		http.Error(w, "Error removing image", http.StatusInternalServerError)
		return
	}

	err = (*dao).RemoveImage(imageID)
	if err != nil {
		http.Error(w, "Error removing image", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Image removed OK..."))
}

func isDevMode() bool {
	runMode := os.Getenv("RUN_MODE")

	return runMode == "dev"
}
