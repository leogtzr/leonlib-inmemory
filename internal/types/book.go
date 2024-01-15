package types

import (
	"fmt"
	"time"
)

// BookInfo ...
type BookInfo struct {
	ID            int
	Title         string
	Author        string
	Description   string
	HasBeenRead   bool
	ImageNames    []string
	Image         []byte
	Base64Images  []BookImageInfo
	AddedOn       string
	GoodreadsLink string
}

type BookSearchType int

const (
	Unknown BookSearchType = iota
	ByTitle
	ByAuthor
)

func (bt BookSearchType) String() string {
	switch bt {
	case ByTitle:
		return "ByTitle"
	case ByAuthor:
		return "ByAuthor"
	default:
		return "Unknown"
	}
}

// BookImageInfo ...
type BookImageInfo struct {
	ImageID int
	BookID  int
	Image   string
}

/*
CREATE TABLE book_likes (

	like_id SERIAL PRIMARY KEY,
	book_id INTEGER REFERENCES books(id),
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	user_id TEXT REFERENCES users(user_id)

);
*/
// BookLike ...
type BookLike struct {
	ID        int
	BookID    int
	CreatedAt time.Time
	UserID    string
}

type Library struct {
	Book []BookInfo
}

func (bi BookInfo) String() string {
	return fmt.Sprintf(`%d) "%s" by "%s"`, bi.ID, bi.Title, bi.Author)
}
