package tests

import (
	"LibAssistant_books/tests/suite"
	"testing"

	booksv1 "github.com/MKode312/protos/gen/go/LibAssistant/books"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddGet_HappyPath(t *testing.T) {
	ctx, st := suite.New(t)

	genre := gofakeit.BookGenre()
	quantity := gofakeit.Number(1, 1000)
	title := gofakeit.BookTitle()

	respAdd, err := st.BooksClient.AddBook(ctx, &booksv1.AddBookRequest{
		Genre:    genre,
		Quantity: int64(quantity),
		Title:    title,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, respAdd.GetBookId())

	respGet, err := st.BooksClient.GetBookByID(ctx, &booksv1.GetBookByIDRequest{
		BookId: respAdd.GetBookId(),
	})
	require.NoError(t, err)

	assert.NotEmpty(t, respGet.GetBook())
}

