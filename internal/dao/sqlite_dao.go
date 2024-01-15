package dao

import (
	"database/sql"
	"encoding/base64"
	"github.com/BurntSushi/toml"
	book "leonlib/internal/types"
	"log"
	"os"
	"path/filepath"
	"time"
)

func addBooksToDatabase(db *sql.DB) error {
	libraryDir := "library"
	libraryDirPath := filepath.Join(libraryDir, "books_db.toml")

	var library book.Library

	if _, err := toml.DecodeFile(libraryDirPath, &library); err != nil {
		return err
	}

	startTime := time.Now()

	for _, book := range library.Book {
		log.Printf("Reading: (%s)", book)

		var bookID int
		stmt, err := db.Prepare("INSERT INTO books(id, title, author, description, read, added_on, goodreads_link) VALUES($1, $2, $3, $4, $5, $6, $7) RETURNING id")
		if err != nil {
			return err
		}

		err = stmt.QueryRow(book.ID, book.Title, book.Author, book.Description, book.HasBeenRead, book.AddedOn, book.GoodreadsLink).Scan(&bookID)
		if err != nil {
			return err
		}

		for _, imageName := range book.ImageNames {
			imgBytes, err := os.ReadFile(filepath.Join("images", imageName))
			if err != nil {
				return err
			}

			imgStmt, err := db.Prepare("INSERT INTO book_images(book_id, image) VALUES($1, $2)")
			if err != nil {
				return err
			}

			_, err = imgStmt.Exec(bookID, imgBytes)
			if err != nil {
				return err
			}
		}
	}

	elapsedTime := time.Since(startTime)

	log.Printf("Books loaded in: %.2f seconds\n", elapsedTime.Seconds())

	return nil
}

func createDB(db *sql.DB) error {
	sqlCommands := []string{
		`CREATE TABLE IF NOT EXISTS books (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			author TEXT NOT NULL,
			description TEXT,
			read BOOLEAN DEFAULT FALSE,
			added_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			goodreads_link TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS book_images (
			image_id INTEGER PRIMARY KEY AUTOINCREMENT,
			book_id INTEGER NOT NULL REFERENCES books(id),
			image BLOB NOT NULL,
			added_on TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			user_id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			name TEXT,
			oauth_identifier TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS book_likes (
			like_id INTEGER PRIMARY KEY AUTOINCREMENT,
			book_id INTEGER REFERENCES books(id),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			user_id TEXT REFERENCES users(user_id),
			UNIQUE(book_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_books_title ON books (title)`,
		`CREATE INDEX IF NOT EXISTS idx_books_author ON books (author)`,
		`CREATE INDEX IF NOT EXISTS idx_books_added_on ON books (added_on)`,
		`CREATE INDEX IF NOT EXISTS idx_book_images_book_id ON book_images (book_id)`,
	}

	for _, sqlCommand := range sqlCommands {
		_, err := db.Exec(sqlCommand)
		if err != nil {
			return err
		}
		log.Printf("SQL command: (%.35s...) executed correctly", sqlCommand)
	}

	return nil
}

func (dao *sqliteBookDAO) AddAll(books []book.BookInfo) error {
	for _, book := range books {
		log.Printf("Reading: (%s)", book)
		bookInfo, err := dao.GetBookByID(book.ID)
		if err == nil && bookInfo.ID == book.ID {
			log.Printf("Book with ID: %d already exists, skipping", book.ID)
			continue
		}

		var bookID int
		stmt, err := dao.db.Prepare("INSERT INTO books(id, title, author, description, read, added_on, goodreads_link) VALUES($1, $2, $3, $4, $5, $6, $7) RETURNING id")
		if err != nil {
			return err
		}

		err = stmt.QueryRow(book.ID, book.Title, book.Author, book.Description, book.HasBeenRead, book.AddedOn, book.GoodreadsLink).Scan(&bookID)
		if err != nil {
			return err
		}

		for _, imageName := range book.ImageNames {
			imgBytes, err := os.ReadFile(filepath.Join("images", imageName))
			if err != nil {
				return err
			}

			imgStmt, err := dao.db.Prepare("INSERT INTO book_images(book_id, image) VALUES($1, $2)")
			if err != nil {
				return err
			}

			_, err = imgStmt.Exec(bookID, imgBytes)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (dao *sqliteBookDAO) AddImageToBook(bookID int, imageData []byte) error {
	if len(imageData) == 0 {
		return nil
	}

	imgStmt, err := dao.db.Prepare("INSERT INTO book_images(book_id, image) VALUES($1, $2)")
	if err != nil {
		return err
	}

	_, err = imgStmt.Exec(bookID, imageData)
	if err != nil {
		return err
	}

	return nil
}

func (dao *sqliteBookDAO) AddUser(userID, email, name, oauthIdentifier string) error {
	_, err := dao.db.Exec(`
			INSERT INTO users(user_id, email, name, oauth_identifier) 
			VALUES($1, $2, $3, $4)
			ON CONFLICT(user_id) DO UPDATE
			SET email = $2, name = $3`, userID, email, name, "Google")

	if err != nil {
		return err
	}

	return nil
}

func (dao *sqliteBookDAO) Close() error {
	return nil
}

func (dao *sqliteBookDAO) CreateBook(book book.BookInfo) error {
	stmt, err := dao.db.Prepare("INSERT INTO books (title, author, image, description, read, goodreads_link) VALUES ($1, $2, $3, $4, $5, $6)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(book.Title, book.Author, book.Image, book.Description, book.HasBeenRead, book.GoodreadsLink)
	if err != nil {
		return err
	}

	return nil
}

func (dao *sqliteBookDAO) GetAllAuthors() ([]string, error) {
	var err error

	allAuthorsRows, err := dao.db.Query("SELECT DISTINCT author FROM books ORDER BY author")
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

func (dao *sqliteBookDAO) GetBookByID(id int) (book.BookInfo, error) {
	var err error
	var queryStr = `SELECT b.id, b.title, b.author, b.description, b.read, b.added_on, b.goodreads_link FROM books b WHERE b.id=$1`

	bookRows, err := dao.db.Query(queryStr, id)
	if err != nil {
		return book.BookInfo{}, err
	}

	defer func() {
		_ = bookRows.Close()
	}()

	var bookInfo book.BookInfo
	var bookID int
	var title string
	var author string
	var description string
	var hasBeenRead bool
	var addedOn time.Time
	var goodreadsLink sql.NullString
	if bookRows.Next() {
		if err := bookRows.Scan(&bookID, &title, &author, &description, &hasBeenRead, &addedOn, &goodreadsLink); err != nil {
			return book.BookInfo{}, err
		}

		bookInfo.ID = bookID
		bookInfo.Title = title
		bookInfo.Author = author
		bookInfo.Description = description
		bookInfo.HasBeenRead = hasBeenRead
		bookInfo.AddedOn = addedOn.Format("2006-01-02")
		if goodreadsLink.Valid {
			bookInfo.GoodreadsLink = goodreadsLink.String
		} else {
			bookInfo.GoodreadsLink = ""
		}
	}

	bookImages, err := dao.GetImagesByBookID(id)
	if err != nil {
		return book.BookInfo{}, err
	}

	bookInfo.Base64Images = bookImages

	return bookInfo, nil
}

func (dao *sqliteBookDAO) GetBookCount() (int, error) {
	rows, err := dao.db.Query(`SELECT count(*) FROM books`)
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

func (dao *sqliteBookDAO) GetBooksWithPagination(offset, limit int) ([]book.BookInfo, error) {
	query := `SELECT id, title, author, description, read, added_on FROM books ORDER BY title LIMIT $1 OFFSET $2;`

	rows, err := dao.db.Query(query, limit, offset)
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

func (dao *sqliteBookDAO) GetBooksBySearchTypeCoincidence(titleSearchText string, bookSearchType book.BookSearchType) ([]book.BookInfo, error) {
	var err error
	queryStr := `SELECT b.id, b.title, b.author, b.description, b.read, b.added_on, b.goodreads_link FROM books b WHERE LOWER(b.title) LIKE '%' || LOWER($1) || '%' ORDER BY b.title`

	if bookSearchType == book.ByAuthor {
		queryStr = `SELECT b.id, b.title, b.author, b.description, b.read, b.added_on, b.goodreads_link FROM books b WHERE LOWER(b.author) LIKE '%' || LOWER($1) || '%' ORDER BY b.title`
	}

	booksByTitleRows, err := dao.db.Query(queryStr, "%"+titleSearchText+"%")
	if err != nil {
		return []book.BookInfo{}, err
	}

	defer booksByTitleRows.Close()

	var books []book.BookInfo
	var id int
	var title string
	var author string
	var description string
	var hasBeenRead bool
	var addedOn time.Time
	var goodreadsLink string
	for booksByTitleRows.Next() {
		var bookInfo book.BookInfo
		if err := booksByTitleRows.Scan(&id, &title, &author, &description, &hasBeenRead, &addedOn, &goodreadsLink); err != nil {
			return []book.BookInfo{}, err
		}

		bookInfo.ID = id
		bookInfo.Title = title
		bookInfo.Author = author
		bookImages, err := dao.GetImagesByBookID(id)
		if err != nil {
			return []book.BookInfo{}, err
		}

		bookInfo.Base64Images = bookImages
		bookInfo.Description = description
		bookInfo.HasBeenRead = hasBeenRead
		bookInfo.AddedOn = addedOn.Format("2006-01-02")
		books = append(books, bookInfo)
	}

	return books, nil
}

func (dao *sqliteBookDAO) GetImagesByBookID(bookID int) ([]book.BookImageInfo, error) {
	bookImagesRows, err := dao.db.Query(`SELECT i.image_id, i.book_id, i.image FROM book_images i WHERE i.book_id=$1`, bookID)
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

func (dao *sqliteBookDAO) LikedBy(bookID, userID string) (bool, error) {
	queryStr := "SELECT EXISTS(SELECT 1 FROM book_likes WHERE book_id=$1 AND user_id=$2)"

	rows, err := dao.db.Query(queryStr, bookID, userID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var exists bool

	if rows.Next() {
		if err := rows.Scan(&exists); err != nil {
			return false, err
		}
	}

	if err != nil {
		return false, err
	}

	return exists, nil
}

func (dao *sqliteBookDAO) LikeBook(bookID, userID string) error {
	_, err := dao.db.Exec("INSERT INTO book_likes(book_id, user_id) VALUES($1, $2) ON CONFLICT(book_id, user_id) DO NOTHING", bookID, userID)

	if err != nil {
		return err
	}

	return nil
}

func (dao *sqliteBookDAO) LikesCount(bookID int) (int, error) {
	var count int
	if err := dao.db.QueryRow("SELECT COUNT(*) FROM book_likes WHERE book_id = $1", bookID).Scan(&count); err != nil {
		return -1, err
	}

	return count, nil
}

func (dao *sqliteBookDAO) Ping() error {
	// TODO: pending...
	return nil
}

func (dao *sqliteBookDAO) RemoveImage(imageID int) error {
	_, err := dao.db.Exec("DELETE FROM book_images WHERE image_id=$1", imageID)
	if err != nil {
		return err
	}

	return nil
}

func (dao *sqliteBookDAO) UnlikeBook(bookID, userID string) error {
	_, err := dao.db.Exec("DELETE FROM book_likes WHERE book_id=$1 AND user_id=$2", bookID, userID)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	return nil
}

func (dao *sqliteBookDAO) UpdateBook(title string, author string, description string, read bool, goodreadsLink string, id int) error {
	bookUpdate, err := dao.db.Prepare(`
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
