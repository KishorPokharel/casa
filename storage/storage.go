package storage

import "database/sql"

type Storage struct {
	Users UserModel
}

func New(db *sql.DB) Storage {
	return Storage{
		Users: UserModel{db},
	}
}
