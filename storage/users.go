package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const timeout = 30 * time.Second

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
	plaintext *string
	hash      []byte
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plaintextPassword
	p.hash = hash
	return nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {

		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err

		}
	}
	return true, nil
}

type UserStorage struct {
	DB *sql.DB
}

func (m *UserStorage) Insert(user User) error {
	query := `
  		insert into users (username, email, password_hash)
  		values ($1, $2, $3) returning id, username, email
	`

	args := []any{user.Username, user.Email, user.Password.hash}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.Username, &user.Email)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}
