package models

type User struct {
	Email    string
	ID       int64
	PassHash []byte
}
