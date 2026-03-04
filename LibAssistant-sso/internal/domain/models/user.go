package models

type User struct {
	Email    string
	ID       string
	PassHash []byte
	IsAdmin  bool
}
