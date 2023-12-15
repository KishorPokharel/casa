package storage

import "database/sql"

type Storage struct {
	Users    UserStorage
	Property PropertyStorage
}

func New(db *sql.DB) Storage {
	return Storage{
		Users:    UserStorage{db},
		Property: PropertyStorage{db},
	}
}
