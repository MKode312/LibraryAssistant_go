package booksgrpc

import (
	"LibAssistant_books/internal/domain/models"
	"context"

	booksv1 "github.com/MKode312/protos/gen/go/LibAssistant/books"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Books interface {
	AddBook(ctx context.Context, genre string, title string, quantity int64) (bookID string, err error)
	TakeBook(ctx context.Context, bookID string, take_copies int64) (success bool, err error)
	GetBook(ctx context.Context, bookID string) (book models.Book, err error)
	DeleteBook(ctx context.Context, bookID string) (success bool, err error)
	GetListOfBooks(ctx context.Context) (books []models.Book, err error)
	FilterBooksByGenreList(ctx context.Context, genre string) (books []models.Book, err error)
}

type serverAPI struct {
	booksv1.UnimplementedBooksServer
	books Books
}

func Register(gRPC *grpc.Server, books Books) {
	booksv1.RegisterBooksServer(gRPC, &serverAPI{books: books})
}

func (s *serverAPI) AddBook(ctx context.Context, req *booksv1.AddBookRequest) (*booksv1.AddBookResponse, error) {
	if err := validateAddBook(req); err != nil {
		return nil, err
	}

	bookID, err := s.books.AddBook(ctx, req.GetGenre(), req.GetTitle(), req.GetQuantity())
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &booksv1.AddBookResponse{
		BookID: bookID,
	}, nil
}

func (s *serverAPI) TakeBook(ctx context.Context, req *booksv1.TakeBookRequest) (*booksv1.TakeBookResponse, error) {
	if err := validateTakeBookFromStore(req); err != nil {
		return nil, err
	}

	success, err := s.books.TakeBook(ctx, req.GetBookID(), req.GetTakeCopies())
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &booksv1.TakeBookResponse{
		Success: success,
	}, nil
}

func (s *serverAPI) GetBook(ctx context.Context, req *booksv1.GetBookRequest) (*booksv1.GetBookResponse, error) {
	if err := validateGetBookFromStore(req); err != nil {
		return nil, err
	}

	book, err := s.books.GetBook(ctx, req.GetBookID())
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &booksv1.GetBookResponse{
		Book: &book.Book,
	}, nil
}

func (s *serverAPI) DeleteBook(ctx context.Context, req *booksv1.DeleteBookRequest) (*booksv1.DeleteBookResponse, error) {
	if err := validateDeleteBookFromStore(req); err != nil {
		return nil, err
	}

	success, err := s.books.DeleteBook(ctx, req.GetBookID())
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &booksv1.DeleteBookResponse{
		Success: success,
	}, nil
}

func (s *serverAPI) GetListOfBooks(ctx context.Context, req *booksv1.GetListOfBooksRequest) (*booksv1.GetListOfBooksResponse, error) {
	books, err := s.books.GetListOfBooks(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	var protoBooks []*booksv1.Book
	for _, book := range books {
		protoBook := &booksv1.Book{
			ID:     book.ID,
			Title:  book.Title,
			AvailableCopies: book.AvailableCopies,
		}
		protoBooks = append(protoBooks, protoBook)
	}

	return &booksv1.GetListOfBooksResponse{
		Books: protoBooks,
	}, nil
}

func (s *serverAPI) FilterBooksByGenreList(ctx context.Context, req *booksv1.FilterBooksByGenreListRequest) (*booksv1.FilterBooksByGenreListResponse, error) {
	if err := validateFilterBooksByGenreListFromStore(req); err != nil {
		return nil, err
	}

	books, err := s.books.FilterBooksByGenreList(ctx, req.GetGenre())
	if err != nil {
		return nil, status.Error(codes.Internal, "internal error")
	}

	var protoBooks []*booksv1.Book
	for _, book := range books {
		protoBook := &booksv1.Book{
			ID: book.ID,
			Title: book.Title,
			Genre: book.Genre,
			AvailableCopies: book.AvailableCopies,
		}
		protoBooks = append(protoBooks, protoBook)
	}

	return &booksv1.FilterBooksByGenreListResponse{
		FilteredBooks: protoBooks,
	}, nil
}

func validateAddBook(req *booksv1.AddBookRequest) error {
	if req.GetGenre() == "" {
		return status.Error(codes.InvalidArgument, "genre is required")
	}

	if req.GetQuantity() == 0 {
		return status.Error(codes.InvalidArgument, "cannot add 0 books")
	}

	if req.GetTitle() == "" {
		return status.Error(codes.InvalidArgument, "title is required")
	}

	return nil
}

func validateTakeBookFromStore(req *booksv1.TakeBookRequest) error {
	if req.GetBookID() == "" {
		return status.Error(codes.InvalidArgument, "book ID is required")
	}

	if req.GetTakeCopies() == 0 {
		return status.Error(codes.InvalidArgument, "cannot take 0 books")
	}

	return nil
}

func validateGetBookFromStore(req *booksv1.GetBookRequest) error {
	if req.GetBookID() == "" {
		return status.Error(codes.InvalidArgument, "book ID is required")
	}

	return nil
}

func validateDeleteBookFromStore(req *booksv1.DeleteBookRequest) error {
	if req.GetBookID() == "" {
		return status.Error(codes.InvalidArgument, "book ID is required")
	}

	return nil
}

func validateFilterBooksByGenreListFromStore(req *booksv1.FilterBooksByGenreListRequest) error {
	if req.GetGenre() == "" {
		return status.Error(codes.InvalidArgument, "genre is required")
	}

	return nil
}
