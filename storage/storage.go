package storage

import "database/sql"

type Storage struct {
	Users UserStorage
}

func New(db *sql.DB) Storage {
	return Storage{
		Users: UserStorage{db},
	}
}
