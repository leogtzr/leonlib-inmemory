package dao

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	book "leonlib/internal/types"
	"time"
)

func (dao *postgresBookDAO) AddAll(books []book.BookInfo) error {
	// TODO: pending...
	return nil
}

func (dao *postgresBookDAO) AddImageToBook(bookID int, imageData []byte) error {
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

func (dao *postgresBookDAO) AddUser(userID, email, name, oauthIdentifier string) error {
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

func (dao *postgresBookDAO) Close() error {
	return nil
}

func (dao *postgresBookDAO) CreateBook(book book.BookInfo) error {
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

func (dao *postgresBookDAO) GetAllAuthors() ([]string, error) {
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

func (dao *postgresBookDAO) GetBookByID(id int) (book.BookInfo, error) {
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

func (dao *postgresBookDAO) GetBookCount() (int, error) {
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

func (dao *postgresBookDAO) GetBooksWithPagination(offset, limit int) ([]book.BookInfo, error) {
	query := `SELECT id, title, author, description, read, added_on FROM books ORDER BY title LIMIT $1 OFFSET $2;`

	fmt.Printf("query=(%s)\n", query)

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

func (dao *postgresBookDAO) GetBooksBySearchTypeCoincidence(titleSearchText string, bookSearchType book.BookSearchType) ([]book.BookInfo, error) {
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

func (dao *postgresBookDAO) GetImagesByBookID(bookID int) ([]book.BookImageInfo, error) {
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

func (dao *postgresBookDAO) LikedBy(bookID, userID string) (bool, error) {
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

func (dao *postgresBookDAO) LikeBook(bookID, userID string) error {
	_, err := dao.db.Exec("INSERT INTO book_likes(book_id, user_id) VALUES($1, $2) ON CONFLICT(book_id, user_id) DO NOTHING", bookID, userID)

	if err != nil {
		return err
	}

	return nil
}

func (dao *postgresBookDAO) LikesCount(bookID int) (int, error) {
	var count int
	if err := dao.db.QueryRow("SELECT COUNT(*) FROM book_likes WHERE book_id = $1", bookID).Scan(&count); err != nil {
		return -1, err
	}

	return count, nil
}

func (dao *postgresBookDAO) Ping() error {
	return dao.db.Ping()
}

func (dao *postgresBookDAO) RemoveImage(imageID int) error {
	_, err := dao.db.Exec("DELETE FROM book_images WHERE image_id=$1", imageID)
	if err != nil {
		return err
	}

	return nil
}

func (dao *postgresBookDAO) UnlikeBook(bookID, userID string) error {
	_, err := dao.db.Exec("DELETE FROM book_likes WHERE book_id=$1 AND user_id=$2", bookID, userID)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	return nil
}

func (dao *postgresBookDAO) UpdateBook(title string, author string, description string, read bool, goodreadsLink string, id int) error {
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
