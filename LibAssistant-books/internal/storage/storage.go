package storage

import "errors"

var (
	ErrBookExists = errors.New("book already exists")
	ErrBookNotFound = errors.New("book not found")
	ErrNothingToList = errors.New("no books in the store")
	ErrNoBooksWithGenre = errors.New("no books in the store with this genre")
	ErrNoCopiesToTake = errors.New("no copies of this book are in the store")
	ErrNotEnoughCopiesInStore = errors.New("not enough copies of this book in the store ")
)