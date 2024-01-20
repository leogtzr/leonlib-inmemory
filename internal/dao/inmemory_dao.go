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
	"strconv"
	"strings"
)

func createInMemoryImagesDatabase(booksDB *map[int]book.BookInfo) (map[int][]book.BookImageInfo, error) {
	//         map[imgID::int][]List of Books
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

func createInMemoryLikesDatabase() *map[string][]string {
	// book_likes[userID::string][]string
	db := make(map[string][]string)

	return &db
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
	likesPerUser := (*dao.bookLikes)[userID]

	return exists(&likesPerUser, bookID), nil
}

func exists(IDs *[]string, target string) bool {
	for _, id := range *IDs {
		if target == id {
			return true
		}
	}

	return false
}

func hasBeenLiked(IDs *[]string, target string) bool {
	for _, id := range *IDs {
		if target == id {
			return true
		}
	}

	return false
}

func (dao *memoryBookDAO) LikeBook(bookID, userID string) error {
	bookLikes, exists := (*dao.bookLikes)[userID]
	if !exists {
		(*dao.bookLikes)[userID] = make([]string, 0)
		(*dao.bookLikes)[userID] = append((*dao.bookLikes)[userID], bookID)

		return nil
	}

	if hasLike := hasBeenLiked(&bookLikes, bookID); !hasLike {
		(*dao.bookLikes)[userID] = append((*dao.bookLikes)[userID], bookID)
	}

	return nil
}

func (dao *memoryBookDAO) LikesCount(bookID int) (int, error) {
	count := 0
	id := strconv.Itoa(bookID)
	for _, bookLikesPerUser := range *dao.bookLikes {
		if exists(&bookLikesPerUser, id) {
			count++
		}
	}

	return count, nil
}

func (dao *memoryBookDAO) Ping() error {
	return nil
}

func (dao *memoryBookDAO) RemoveImage(imageID int) error {
	delete(*dao.images, imageID)

	return nil
}

func removeIndex(elements []string, index int) []string {
	ret := make([]string, 0)
	ret = append(ret, elements[:index]...)

	return append(ret, elements[index+1:]...)
}

func find(elements []string, target string) int {
	foundIndex := -1

	for i, e := range elements {
		if target == e {
			foundIndex = i

			break
		}
	}

	return foundIndex
}

func (dao *memoryBookDAO) UnlikeBook(bookID, userID string) error {
	// Remove the like made by the user
	bookLikes, exists := (*dao.bookLikes)[userID]
	if !exists {
		return fmt.Errorf("error: user (%s) does not have liked books", userID)
	}

	bookIDxToRemove := find(bookLikes, bookID)
	if bookIDxToRemove == -1 {
		return fmt.Errorf("error: user (%s) does not have liked books", userID)
	}

	(*dao.bookLikes)[userID] = removeIndex(bookLikes, bookIDxToRemove)

	return nil
}

func (dao *memoryBookDAO) UpdateBook(title string, author string, description string, read bool, goodreadsLink string, id int) error {
	book := (*dao.books)[id]
	book.Title = title
	book.Author = author
	book.Description = description
	book.HasBeenRead = read
	book.GoodreadsLink = goodreadsLink

	(*dao.books)[id] = book
	
	return nil
}
