package models

type Book struct {
	ID              string
	Title           string
	Genre           string
	AvailableCopies int64
}

type Student struct {
	ID       string
	FullName string
	Grade    int32
	Letter   string
}
