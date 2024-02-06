package dao

import (
	"database/sql"
	book "leonlib/internal/types"
	user "leonlib/internal/types"
	"log"
	"os"
	"path/filepath"
	"time"
)

func (dao *postgresBookDAO) AddAll(books []book.BookInfo) error {
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

func (dao *postgresBookDAO) AddImageToBook(bookID int, imageData []byte) error {
	return addImageToBook(bookID, imageData, dao.db)
}

func (dao *postgresBookDAO) AddUser(userID, email, name, oauthIdentifier string) error {
	return addUser(dao.db, userID, email, name, oauthIdentifier)
}

func (dao *postgresBookDAO) Close() error {
	return nil
}

func (dao *postgresBookDAO) CreateBook(book book.BookInfo) error {
	stmt, err := dao.db.Prepare("INSERT INTO books (title, author, description, read, goodreads_link) VALUES ($1, $2, $3, $4, $5)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	insertedBookIDResult, err := stmt.Exec(book.Title, book.Author, book.Description, book.HasBeenRead, book.GoodreadsLink)
	if err != nil {
		return err
	}

	insertedBookID, err := insertedBookIDResult.LastInsertId()
	if err != nil {
		return err
	}

	imgStmt, err := dao.db.Prepare("INSERT INTO book_images(book_id, image) VALUES($1, $2)")
	if err != nil {
		return err
	}

	_, err = imgStmt.Exec(insertedBookID, book.Image)
	if err != nil {
		return err
	}

	return nil
}

func (dao *postgresBookDAO) GetAllAuthors() ([]string, error) {
	return getAllAuthors(dao.db)
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
	return getBookCount(dao.db)
}

func (dao *postgresBookDAO) GetBooksWithPagination(offset, limit int) ([]book.BookInfo, error) {
	return getBooksWithPagination(offset, limit, dao.db)
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
	return getImagesByBookID(bookID, dao.db)
}

func (dao *postgresBookDAO) GetUserInfoByID(id string) (user.UserInfo, error) {
	var err error
	var queryStr = `SELECT u.user_id, u.email, u.name FROM users u WHERE u.user_id=$1`

	userRow, err := dao.db.Query(queryStr, id)
	if err != nil {
		return user.UserInfo{}, err
	}

	defer func() {
		_ = userRow.Close()
	}()

	var userInfo user.UserInfo
	var userID string
	var email string
	var name string
	if userRow.Next() {
		if err := userRow.Scan(&userID, &email, &name); err != nil {
			return user.UserInfo{}, err
		}

		userInfo.Sub = userID
		userInfo.Email = email
		userInfo.Name = name
	}

	return userInfo, nil
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
	if _, err := dao.db.Exec("DELETE FROM book_images WHERE image_id=$1", imageID); err != nil {
		return err
	}

	return nil
}

func (dao *postgresBookDAO) UnlikeBook(bookID, userID string) error {
	if _, err := dao.db.Exec("DELETE FROM book_likes WHERE book_id=$1 AND user_id=$2", bookID, userID); err != nil {
		return err
	}

	return nil
}

func (dao *postgresBookDAO) UpdateBook(title string, author string, description string, read bool, goodreadsLink string, id int) error {
	return updateBook(title, author, description, read, goodreadsLink, id, dao.db)
}
