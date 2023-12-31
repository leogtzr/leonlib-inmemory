package types

import "fmt"

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

type BookImageInfo struct {
	ImageID int
	BookID  int
	Image   string
}

type Library struct {
	Book []BookInfo
}

func (bi BookInfo) String() string {
	return fmt.Sprintf(`%d) "%s" by "%s"`, bi.ID, bi.Title, bi.Author)
}
