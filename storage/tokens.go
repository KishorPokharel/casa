package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"time"
)

const (
	ScopeActivation    = "activation"
	ScopePasswordReset = "password-reset"
)

type Token struct {
	PlainText string
	Hash      []byte
	UserID    int64
	Expiry    time.Time
	Scope     string
}

func generateToken(userId int64, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userId,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}
	token.PlainText = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.PlainText))
	token.Hash = hash[:]

	return token, nil
}

type TokenStorage struct {
	DB *sql.DB
}

func (s *TokenStorage) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = s.Insert(token)
	return token, err
}

func (s *TokenStorage) Insert(token *Token) error {
	query := `
		insert into tokens (hash, user_id, expiry, scope)
		values ($1, $2, $3, $4)`

	args := []any{token.Hash, token.UserID, token.Expiry, token.Scope}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := s.DB.ExecContext(ctx, query, args...)
	return err
}

func (s *TokenStorage) DeleteAllForUser(scope string, userID int64) error {
	query := `
        delete from tokens
        where scope = $1 and user_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := s.DB.ExecContext(ctx, query, scope, userID)
	return err
}
