package booksgrpc

import (
	"LibAssistant_books/internal/domain/models"
	"LibAssistant_books/internal/lib/convert"
	"LibAssistant_books/internal/services/books"
	"context"
	"errors"

	booksv1 "github.com/MKode312/protos/gen/go/LibAssistant/books"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Books interface {
	AddBook(ctx context.Context, genre string, title string, quantity int64) (bookID string, err error)
	AddCopies(ctx context.Context, bookID string, copiesToAdd int64) (success bool, err error)
	TakeBook(ctx context.Context, bookID string, take_copies int64) (success bool, err error)
	GetBookByID(ctx context.Context, bookID string) (book models.Book, err error)
	GetBookByTitle(ctx context.Context, title string) (book models.Book, err error)
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
		if errors.Is(err, books.ErrBookExists) {
			return nil, status.Error(codes.AlreadyExists, books.ErrBookExists.Error())
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &booksv1.AddBookResponse{
		BookId: bookID,
	}, nil
}

func (s *serverAPI) AddCopies(ctx context.Context, req *booksv1.AddCopiesRequest) (*booksv1.AddCopiesResponse, error) {
	if err := validateAddCopies(req); err != nil {
		return nil, err
	}

	success, err := s.books.AddCopies(ctx, req.GetBookId(), req.GetCopiesToAdd())
	if err != nil {
		if errors.Is(err, books.ErrBookNotFound) {
			return nil, status.Error(codes.NotFound, books.ErrBookNotFound.Error())
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &booksv1.AddCopiesResponse{
		Success: success,
	}, nil
}

func (s *serverAPI) TakeBook(ctx context.Context, req *booksv1.TakeBookRequest) (*booksv1.TakeBookResponse, error) {
	if err := validateTakeBook(req); err != nil {
		return nil, err
	}

	success, err := s.books.TakeBook(ctx, req.GetBookId(), req.GetTakeCopies())
	if err != nil {
		if errors.Is(err, books.ErrBookNotFound) {
			return nil, status.Error(codes.NotFound, books.ErrBookNotFound.Error())
		}
		if errors.Is(err, books.ErrNoCopiesToTake) {
			return nil, status.Error(codes.OutOfRange, books.ErrNoCopiesToTake.Error())
		}
		if errors.Is(err, books.ErrNotEnoughCopiesInStore) {
			return nil, status.Error(codes.FailedPrecondition, books.ErrNotEnoughCopiesInStore.Error())
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &booksv1.TakeBookResponse{
		Success: success,
	}, nil
}

func (s *serverAPI) GetBookByID(ctx context.Context, req *booksv1.GetBookByIDRequest) (*booksv1.GetBookByIDResponse, error) {
	if err := validateGetBookByID(req); err != nil {
		return nil, err
	}

	book, err := s.books.GetBookByID(ctx, req.GetBookId())
	if err != nil {
		if errors.Is(err, books.ErrBookNotFound) {
			return nil, status.Error(codes.NotFound, books.ErrBookNotFound.Error())
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &booksv1.GetBookByIDResponse{
		Book: convert.ToProto(&book),
	}, nil
}

func (s *serverAPI) GetBookByTitle(ctx context.Context, req *booksv1.GetBookByTitleRequest) (*booksv1.GetBookByTitleResponse, error) {
	if err := validateGetBookByTitle(req); err != nil {
		return nil, err
	}

	book, err := s.books.GetBookByTitle(ctx, req.GetTitle())
	if err != nil {
		if errors.Is(err, books.ErrBookNotFound) {
			return nil, status.Error(codes.NotFound, books.ErrBookNotFound.Error())
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &booksv1.GetBookByTitleResponse{
		Book: convert.ToProto(&book),
	}, nil
}

func (s *serverAPI) DeleteBook(ctx context.Context, req *booksv1.DeleteBookRequest) (*booksv1.DeleteBookResponse, error) {
	if err := validateDeleteBook(req); err != nil {
		return nil, err
	}

	success, err := s.books.DeleteBook(ctx, req.GetBookId())
	if err != nil {
		if errors.Is(err, books.ErrBookNotFound) {
			return nil, status.Error(codes.NotFound, books.ErrBookNotFound.Error())
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &booksv1.DeleteBookResponse{
		Success: success,
	}, nil
}

func (s *serverAPI) GetListOfBooks(ctx context.Context, req *booksv1.GetListOfBooksRequest) (*booksv1.GetListOfBooksResponse, error) {
	booksList, err := s.books.GetListOfBooks(ctx)
	if err != nil {
		if errors.Is(err, books.ErrNothingToList) {
			return nil, status.Error(codes.OutOfRange, books.ErrNothingToList.Error())
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	var protoBooks []*booksv1.Book
	for _, book := range booksList {
		protoBook := &booksv1.Book{
			ID:              book.ID,
			Title:           book.Title,
			Genre:           book.Genre,
			AvailableCopies: book.AvailableCopies,
		}
		protoBooks = append(protoBooks, protoBook)
	}

	return &booksv1.GetListOfBooksResponse{
		Books: protoBooks,
	}, nil
}

func (s *serverAPI) FilterBooksByGenreList(ctx context.Context, req *booksv1.FilterBooksByGenreListRequest) (*booksv1.FilterBooksByGenreListResponse, error) {
	if err := validateFilterBooksByGenreList(req); err != nil {
		return nil, err
	}

	booksList, err := s.books.FilterBooksByGenreList(ctx, req.GetGenre())
	if err != nil {
		if errors.Is(err, books.ErrNoBooksWithGenre) {
			return nil, status.Error(codes.NotFound, books.ErrNoBooksWithGenre.Error())
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	var protoBooks []*booksv1.Book
	for _, book := range booksList {
		protoBook := &booksv1.Book{
			ID:              book.ID,
			Title:           book.Title,
			Genre:           book.Genre,
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
		return status.Error(codes.InvalidArgument, "cannot add 0 copies")
	}

	if req.GetTitle() == "" {
		return status.Error(codes.InvalidArgument, "title is required")
	}

	return nil
}

func validateAddCopies(req *booksv1.AddCopiesRequest) error {
	if req.GetBookId() == "" {
		return status.Error(codes.InvalidArgument, "book ID is required")
	}
	if req.GetCopiesToAdd() == 0 {
		return status.Error(codes.InvalidArgument, "cannot add 0 copies")
	}

	return nil
}

func validateTakeBook(req *booksv1.TakeBookRequest) error {
	if req.GetBookId() == "" {
		return status.Error(codes.InvalidArgument, "book ID is required")
	}

	if req.GetTakeCopies() == 0 {
		return status.Error(codes.InvalidArgument, "cannot take 0 copies")
	}

	return nil
}

func validateGetBookByID(req *booksv1.GetBookByIDRequest) error {
	if req.GetBookId() == "" {
		return status.Error(codes.InvalidArgument, "book ID is required")
	}

	return nil
}

func validateGetBookByTitle(req *booksv1.GetBookByTitleRequest) error {
	if req.GetTitle() == "" {
		return status.Error(codes.InvalidArgument, "book title is required")
	}

	return nil
}

func validateDeleteBook(req *booksv1.DeleteBookRequest) error {
	if req.GetBookId() == "" {
		return status.Error(codes.InvalidArgument, "book ID is required")
	}

	return nil
}

func validateFilterBooksByGenreList(req *booksv1.FilterBooksByGenreListRequest) error {
	if req.GetGenre() == "" {
		return status.Error(codes.InvalidArgument, "genre is required")
	}

	return nil
}
