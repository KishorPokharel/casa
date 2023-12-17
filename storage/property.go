package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Property struct {
	ID          int64
	UserID      int64
	Banner      string
	Location    string
	Title       string
	Description string
	Price       int64
	CreatedAt   time.Time
}

type PropertyStorage struct {
	DB *sql.DB
}

func (s *PropertyStorage) GetAll() ([]Property, error) {
	query := `
        select id, title, description, banner, location, price, created_at
        from listings
    `

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	listings := []Property{}
	for rows.Next() {
		p := Property{}
		err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Banner, &p.Location, &p.Price, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		listings = append(listings, p)
	}
	return listings, nil
}

func (s *PropertyStorage) Insert(property Property) error {
	query := `
      insert into listings
      (title, user_id, description, banner, location, property_type, price)
      values ($1, $2, $3, $4, $5, $6, $7)
    `
	args := []any{property.Title, property.UserID, property.Description, property.Banner, property.Location, "land", property.Price}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	return nil
}

func (s *PropertyStorage) Search(searchQuery string) ([]Property, error) {
	query := `
        select id, title, description, banner, location, price, created_at
        from listings
        where (to_tsvector('simple', location) @@ plainto_tsquery('simple', $1) or $1='')
    `
	args := []any{searchQuery}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	listings := []Property{}
	for rows.Next() {
		p := Property{}
		err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Banner, &p.Location, &p.Price, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		listings = append(listings, p)
	}
	return listings, nil
}

func (s *PropertyStorage) Get(id int64) (Property, error) {
	query := `
        select id, title, description, banner, location, price, created_at
        from listings
        where id = $1
    `

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	row := s.DB.QueryRowContext(ctx, query, id)
	p := Property{}
	err := row.Scan(&p.ID, &p.Title, &p.Description, &p.Banner, &p.Location, &p.Price, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return p, ErrNoRecord
		}
		return p, err
	}
	return p, nil
}
