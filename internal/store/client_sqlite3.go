package store

import _ "github.com/mattn/go-sqlite3" // third party driver sql3

func NewInMemorySQLiteClient() (*Client, error) {
	return Open("sqlite3", "file:chat-service?mode=memory&cache=shared&_fk=1")
}
