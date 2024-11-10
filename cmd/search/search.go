package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	model "leonlib/internal/types"
)

func createInMemoryDatabaseFromFile() (map[int]model.BookInfo, error) {
	libraryDir := "library"
	libraryDirPath := filepath.Join(libraryDir, "books_db.toml")

	var library model.Library

	if _, err := toml.DecodeFile(libraryDirPath, &library); err != nil {
		return nil, err
	}

	db := make(map[int]model.BookInfo)
	for _, book := range library.Book {
		db[book.ID] = book
	}
	return db, nil
}

func searchBooks(books map[int]model.BookInfo, query string, searchByTitle, searchByAuthor bool) {
	matchesCount := 0
	query = strings.ToLower(query)
	for _, book := range books {
		match := false
		if searchByTitle && strings.Contains(strings.ToLower(book.Title), query) {
			match = true
		}
		if searchByAuthor && strings.Contains(strings.ToLower(book.Author), query) {
			match = true
		}
		if match {
			fmt.Printf("\"%s\" by %s\n", book.Title, book.Author)
			if book.Description != "" {
				fmt.Printf("%s\n", book.Description)
			}
			fmt.Printf("id: %d\n", book.ID)
			fmt.Printf("Agregado el: %s\n", book.AddedOn)
			if book.HasBeenRead {
				fmt.Println("Leído: sí")
			} else {
				fmt.Println("Leído: no")
			}
			fmt.Println()
			matchesCount++
		}
	}

	fmt.Printf("%d books found (from %d books documented).\n", matchesCount, len(books))
}

func main() {
	titleFlag := flag.String("title", "", "Buscar por título")
	authorFlag := flag.String("author", "", "Buscar por autor")
	flag.Parse()

	query := ""
	if len(flag.Args()) > 0 {
		query = flag.Args()[0]
	}

	books, err := createInMemoryDatabaseFromFile()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error al cargar la base de datos: %v\n", err)
		os.Exit(1)
	}

	if *titleFlag != "" {
		searchBooks(books, *titleFlag, true, false)
	} else if *authorFlag != "" {
		searchBooks(books, *authorFlag, false, true)
	} else if query != "" {
		searchBooks(books, query, true, true)
	} else {
		fmt.Println("Uso: search [-title \"model title\"] [-author \"author\"] [\"query\"]")
	}
}
