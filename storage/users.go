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
	Phone     string
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

func (s *UserStorage) Insert(user User) error {
	query := `
  		insert into users (username, email, password_hash)
  		values ($1, $2, $3) returning id, username, email
	`

	args := []any{user.Username, user.Email, user.Password.hash}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := s.DB.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.Username, &user.Email)
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

func (s *UserStorage) Authenticate(email, password string) (int64, error) {
	query := `select id, password_hash
      from users where email = $1
    `
	args := []any{email}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var id int64
	var hashedPassword []byte
	err := s.DB.QueryRowContext(ctx, query, args...).Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}
	return id, nil
}

func (s *UserStorage) Exists(id int64) (bool, error) {
	var exists bool
	query := "select exists(select true from users where id = $1)"
	err := s.DB.QueryRow(query, id).Scan(&exists)
	return exists, err
}

func (s *UserStorage) Get(id int64) (User, error) {
	query := `select id, username, email, phone, created_at from users where id = $1`

	var user User
	var phone sql.Null[string]
	err := s.DB.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.Email, &phone, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNoRecord
		} else {
			return User{}, err
		}
	}
	if phone.Valid {
		user.Phone = phone.V
	} else {
		user.Phone = ""
	}

	return user, nil
}
