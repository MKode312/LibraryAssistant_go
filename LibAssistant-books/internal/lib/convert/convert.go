package convert

import (
	"LibAssistant_books/internal/domain/models"

	booksv1 "github.com/MKode312/protos/gen/go/LibAssistant/books"
)

func ToDomain(protoBook *booksv1.Book) *models.Book {
	return &models.Book{
		ID: protoBook.GetID(),
		Title: protoBook.GetTitle(),
		Genre: protoBook.GetGenre(),
		AvailableCopies: protoBook.GetAvailableCopies(),
	}
}

func ToProto(book *models.Book) *booksv1.Book {
	return &booksv1.Book{
		ID: book.ID,
		Title: book.Title,
		Genre: book.Genre,
		AvailableCopies: book.AvailableCopies,
	}
}