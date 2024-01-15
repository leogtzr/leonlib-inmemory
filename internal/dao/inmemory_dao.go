package dao

import (
	"encoding/base64"
	"fmt"
	"github.com/BurntSushi/toml"
	book "leonlib/internal/types"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func createInMemoryImagesDatabase(booksDB *map[int]book.BookInfo) (map[int][]book.BookImageInfo, error) {
	db := make(map[int][]book.BookImageInfo)

	imageID := 0
	for _, b := range *booksDB {
		var images []book.BookImageInfo
		for _, imageName := range b.ImageNames {
			log.Printf("debug:x Reading image name=(%s) for book=(%s)\n", imageName, b)
			imgBytes, err := os.ReadFile(filepath.Join("images", imageName))
			if err != nil {
				return map[int][]book.BookImageInfo{}, err
			}

			if len(imgBytes) > 0 {
				encodedImage := base64.StdEncoding.EncodeToString(imgBytes)
				bookImageInfo := book.BookImageInfo{
					ImageID: imageID,
					BookID:  b.ID,
					Image:   encodedImage,
				}
				images = append(images, bookImageInfo)
			}

			db[b.ID] = images

			imageID++
		}
	}

	return db, nil
}

func createInMemoryDatabaseFromFile() (map[int]book.BookInfo, error) {
	libraryDir := "library"
	libraryDirPath := filepath.Join(libraryDir, "books_db.toml")

	var library book.Library

	if _, err := toml.DecodeFile(libraryDirPath, &library); err != nil {
		return map[int]book.BookInfo{}, err
	}

	db := make(map[int]book.BookInfo)

	for _, book := range library.Book {
		db[book.ID] = book
	}

	return db, nil
}

func searchByTitle(titleSearchText string, db *map[int]book.BookInfo) (*[]book.BookInfo, error) {
	if len(titleSearchText) == 0 {
		return &[]book.BookInfo{}, fmt.Errorf("title search text empty")
	}

	var results []book.BookInfo
	for _, bookInfo := range *db {
		title := strings.ToLower(bookInfo.Title)
		has := strings.Contains(title, strings.ToLower(titleSearchText))
		if has {
			results = append(results, bookInfo)
		}
	}

	return &results, nil
}

func searchByAuthor(authorSearchText string, db *map[int]book.BookInfo) (*[]book.BookInfo, error) {
	if len(authorSearchText) == 0 {
		return &[]book.BookInfo{}, fmt.Errorf("author search text empty")
	}
	var results []book.BookInfo
	for _, bookInfo := range *db {
		author := strings.ToLower(bookInfo.Author)
		if has := strings.Contains(author, strings.ToLower(authorSearchText)); has {
			results = append(results, bookInfo)
		}
	}

	return &results, nil
}

func (dao *memoryBookDAO) AddAll(books []book.BookInfo) error {
	return nil
}

func (dao *memoryBookDAO) AddImageToBook(bookID int, imageData []byte) error {
	if len(imageData) == 0 {
		return nil
	}

	// TODO: pending...

	return nil
}

func (dao *memoryBookDAO) AddUser(userID, email, name, oauthIdentifier string) error {
	// TODO: pending...

	return nil
}

func (dao *memoryBookDAO) Close() error {
	fmt.Printf("debug:x pending impl 4")
	return nil
}

func (dao *memoryBookDAO) CreateBook(book book.BookInfo) error {
	fmt.Printf("debug:x pending impl 2")
	return nil
}

func (dao *memoryBookDAO) GetAllAuthors() ([]string, error) {
	authors := map[string]struct{}{}

	for _, v := range *dao.books {
		authors[v.Author] = struct{}{}
	}

	authorsUnique := make([]string, 0, len(authors))

	for k, _ := range authors {
		authorsUnique = append(authorsUnique, k)
	}

	sort.Strings(authorsUnique)

	return authorsUnique, nil
}

func (dao *memoryBookDAO) GetBookByID(id int) (book.BookInfo, error) {
	bookInfo, ok := (*dao.books)[id]
	if !ok {
		return book.BookInfo{}, fmt.Errorf("id %d does not exist", id)
	}

	bookImages, err := dao.GetImagesByBookID(id)
	if err != nil {
		return book.BookInfo{}, err
	}

	bookInfo.Base64Images = bookImages

	return bookInfo, nil
}

func (dao *memoryBookDAO) GetBookCount() (int, error) {
	return len(*dao.books), nil
}

func (dao *memoryBookDAO) GetBooksWithPagination(offset, limit int) ([]book.BookInfo, error) {
	if offset > len(*dao.books) {
		offset = len(*dao.books)
	}

	end := offset + limit
	if end > len(*dao.books) {
		end = len(*dao.books)
	}

	var books []book.BookInfo

	for _, bookInfo := range *dao.books {
		books = append(books, bookInfo)
	}

	sort.Slice(books, func(i, j int) bool {
		return books[i].Title < books[j].Title
	})

	return books[offset:end], nil
}

func (dao *memoryBookDAO) GetBooksBySearchTypeCoincidence(titleSearchText string, bookSearchType book.BookSearchType) ([]book.BookInfo, error) {
	var err error

	var found *[]book.BookInfo

	switch bookSearchType {
	case book.ByAuthor:
		found, err = searchByAuthor(titleSearchText, dao.books)
		if err != nil {
			return []book.BookInfo{}, err
		}
	case book.ByTitle:
		found, err = searchByTitle(titleSearchText, dao.books)
		if err != nil {
			return []book.BookInfo{}, err
		}
	}

	for i := range *found {
		bookInfo := &(*found)[i]
		bookImages, err := dao.GetImagesByBookID(bookInfo.ID)
		if err != nil {
			return []book.BookInfo{}, err
		}

		bookInfo.Base64Images = bookImages
	}

	return *found, nil
}

func (dao *memoryBookDAO) GetImagesByBookID(bookID int) ([]book.BookImageInfo, error) {
	images, ok := (*dao.images)[bookID]
	if !ok {
		return []book.BookImageInfo{}, nil
	}

	return images, nil
}

func (dao *memoryBookDAO) LikedBy(bookID, userID string) (bool, error) {
	return false, nil
}

func (dao *memoryBookDAO) LikeBook(bookID, userID string) error {
	// TODO: pending...

	return nil
}

func (dao *memoryBookDAO) LikesCount(bookID int) (int, error) {
	// TODO: pending...
	return -1, nil
}

func (dao *memoryBookDAO) Ping() error {
	// TODO: pending....
	return nil
}

func (dao *memoryBookDAO) RemoveImage(imageID int) error {
	// TODO: pending...
	return nil
}

func (dao *memoryBookDAO) UnlikeBook(bookID, userID string) error {
	// TODO: pending...

	return nil
}

func (dao *memoryBookDAO) UpdateBook(title string, author string, description string, read bool, goodreadsLink string, id int) error {
	// TODO: pending
	return nil
}
