package storage

import "database/sql"

type User struct {
	ID       int64
	Username string
	Email    string
	Password password
}

type password struct {
	Hash      []byte
	PlainText string
}

type UserModel struct {
	DB *sql.DB
}

func (m *UserModel) Insert(user User) error {
	query := `
  		insert into users (username, email, password)
  		values ($1, $2, $3) returning (id, username, email)
	`

	args := []any{user.Username, user.Email, user.Password.Hash}
	row := m.DB.QueryRow(query, args...)
	return row.Scan(&user.ID, &user.Username, &user.Email)
}
