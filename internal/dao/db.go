package dao

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	book "leonlib/internal/types"
)

// DAO
type DAO interface {
	AddAll([]book.BookInfo) error
	AddImageToBook(bookID int, imageData []byte) error
	AddUser(userID, email, name, oauthIdentifier string) error
	Close() error
	CreateBook(book book.BookInfo) error
	GetAllAuthors() ([]string, error)
	GetBookByID(id int) (book.BookInfo, error)
	GetBookCount() (int, error)
	GetBooksWithPagination(offset, limit int) ([]book.BookInfo, error)
	GetBooksBySearchTypeCoincidence(titleSearchText string, bookSearchType book.BookSearchType) ([]book.BookInfo, error)
	GetImagesByBookID(bookID int) ([]book.BookImageInfo, error)
	LikedBy(bookID, userID string) (bool, error)
	LikeBook(bookID, userID string) error
	LikesCount(bookID int) (int, error)
	Ping() error
	RemoveImage(imageID int) error
	UnlikeBook(bookID, userID string) error
	UpdateBook(title string, author string, description string, read bool, goodreadsLink string, id int) error
}

type sqliteBookDAO struct {
	db *sql.DB
}

type postgresBookDAO struct {
	db *sql.DB
}

type memoryBookDAO struct {
	books     *map[int]book.BookInfo
	images    *map[int][]book.BookImageInfo
	bookLikes *map[string][]string
}

func NewDAO(dbMode, dbHost, dbPort, dbUser, dbPassword, dbName string) (DAO, error) {
	var bookDAO DAO
	switch dbMode {
	case "inmemory":
		DB, err := sql.Open("sqlite3", "/var/lib/appdata/leonlib.db")
		if err != nil {
			return nil, err
		}
		bookDAO = &sqliteBookDAO{db: DB}
		err = createDB(DB)
		if err != nil {
			return nil, err
		}
		err = addBooksToDatabase(DB)
		if err != nil {
			return nil, err
		}

	case "postgres":
		var psqlInfo string

		psqlInfo = "host=" + dbHost + " port=" + dbPort + " user=" + dbUser + " password=" + dbPassword + " dbname=" + dbName + " sslmode=disable"

		fmt.Printf("debug:x connection=(%s)\n", psqlInfo)

		DB, err := sql.Open("postgres", psqlInfo)
		if err != nil {
			return nil, err
		}
		bookDAO = &postgresBookDAO{db: DB}

	case "memory":
		db, err := createInMemoryDatabaseFromFile()
		if err != nil {
			return nil, err
		}
		images, err := createInMemoryImagesDatabase(&db)
		if err != nil {
			return nil, err
		}

		bookDAO = &memoryBookDAO{books: &db, images: &images, bookLikes: createInMemoryLikesDatabase()}
	}

	return bookDAO, nil
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

func addImageToBook(bookID int, imageData []byte, db *sql.DB) error {
	if len(imageData) == 0 {
		return nil
	}

	imgStmt, err := db.Prepare("INSERT INTO book_images(book_id, image) VALUES($1, $2)")
	if err != nil {
		return err
	}

	_, err = imgStmt.Exec(bookID, imageData)
	if err != nil {
		return err
	}

	return nil
}

func getBookCount(db *sql.DB) (int, error) {
	rows, err := db.Query(`SELECT count(*) FROM books`)
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

func getBooksWithPagination(offset, limit int, db *sql.DB) ([]book.BookInfo, error) {
	query := `SELECT id, title, author, description, read, added_on FROM books ORDER BY title LIMIT $1 OFFSET $2;`

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

func addUser(db *sql.DB, userID, email, name, oauthIdentifier string) error {
	_, err := db.Exec(`
			INSERT INTO users(user_id, email, name, oauth_identifier) 
			VALUES($1, $2, $3, $4)
			ON CONFLICT(user_id) DO UPDATE
			SET email = $2, name = $3`, userID, email, name, "Google")

	if err != nil {
		return err
	}

	return nil
}

func getImagesByBookID(bookID int, db *sql.DB) ([]book.BookImageInfo, error) {
	bookImagesRows, err := db.Query(`SELECT i.image_id, i.book_id, i.image FROM book_images i WHERE i.book_id=$1`, bookID)
	if err != nil {
		return []book.BookImageInfo{}, err
	}

	defer func() {
		_ = bookImagesRows.Close()
	}()

	var images []book.BookImageInfo

	for bookImagesRows.Next() {
		var imageID int
		var bookID int
		var base64Image []byte
		if err = bookImagesRows.Scan(&imageID, &bookID, &base64Image); err != nil {
			return []book.BookImageInfo{}, err
		}

		if len(base64Image) > 0 {
			encodedImage := base64.StdEncoding.EncodeToString(base64Image)
			bookImageInfo := book.BookImageInfo{
				ImageID: imageID,
				BookID:  bookID,
				Image:   encodedImage,
			}
			images = append(images, bookImageInfo)
		}
	}

	return images, nil
}

func updateBook(title string, author string, description string, read bool, goodreadsLink string, id int, db *sql.DB) error {
	bookUpdate, err := db.Prepare(`
		UPDATE books SET 
			title = $1,
			author = $2,
			description = $3,
			read = $4,
			goodreads_link = $5
		WHERE id = $6
	`)
	if err != nil {
		return err
	}
	defer func() {
		_ = bookUpdate.Close()
	}()

	_, err = bookUpdate.Exec(title, author, description, read, goodreadsLink, id)
	if err != nil {
		return err
	}

	return nil
}
