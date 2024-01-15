package dao

import (
	"database/sql"
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
	books  *map[int]book.BookInfo
	images *map[int][]book.BookImageInfo
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

		bookDAO = &memoryBookDAO{books: &db, images: &images}
	}

	return bookDAO, nil
}
