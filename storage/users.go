package storage

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrNoRecord           = errors.New("no record found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrDuplicateEmail     = errors.New("duplicate email")
)

type User struct {
	ID        int64
	Username  string
	Email     string
	Password  password
	CreatedAt time.Time
}

type password struct {
	Hash      []byte
	PlainText string
}

type UserStorage struct {
	DB *sql.DB
}

func (m *UserStorage) Insert(user User) error {
	query := `
  		insert into users (username, email, password)
  		values ($1, $2, $3) returning (id, username, email)
	`

	args := []any{user.Username, user.Email, user.Password.Hash}
	row := m.DB.QueryRow(query, args...)
	return row.Scan(&user.ID, &user.Username, &user.Email)
}
