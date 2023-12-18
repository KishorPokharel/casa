package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
)

var ErrDuplicateSave = errors.New("listing is already saved")

type Property struct {
	ID          int64
	UserID      int64
	Banner      string
	Location    string
	Title       string
	Description string
	Price       int64
	Username    string
	CreatedAt   time.Time
	Rank        float64
}

type PropertyStorage struct {
	DB *sql.DB
}

func (s *PropertyStorage) GetAll() ([]Property, error) {
	query := `
        select 
            listings.id, listings.title, listings.description, listings.banner, listings.location,
            listings.price, listings.created_at, users.id, users.username
        from
            listings
        join
            users on listings.user_id = users.id
        order by listings.created_at desc
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
		err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Banner, &p.Location, &p.Price, &p.CreatedAt, &p.UserID, &p.Username)
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
        select
            listings.id, listings.title, listings.description, listings.banner, listings.location,
            listings.price, listings.created_at, users.id, users.username,
            ts_rank(to_tsvector('simple', listings.location), plainto_tsquery($1)) * 3 +
            ts_rank(to_tsvector('simple', listings.title), plainto_tsquery($1)) * 2 +
            ts_rank(to_tsvector('simple', listings.description), plainto_tsquery($1)) * 1 as rank
        from
            listings
        join
            users on listings.user_id = users.id
        where
            (to_tsvector('simple', location || ' ' || description || ' ' || title) @@ plainto_tsquery('simple', $1) or $1='')
        order by rank desc
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
		err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Banner, &p.Location, &p.Price, &p.CreatedAt, &p.UserID, &p.Username, &p.Rank)
		if err != nil {
			return nil, err
		}
		listings = append(listings, p)
	}
	return listings, nil
}

func (s *PropertyStorage) Get(id int64) (Property, error) {
	query := `
        select
            listings.id, listings.title, listings.description, listings.banner, listings.location,
            listings.price, listings.created_at, users.id, users.username
        from
            listings
        join
            users on listings.user_id = users.id
        where listings.id = $1
    `

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	row := s.DB.QueryRowContext(ctx, query, id)
	p := Property{}
	err := row.Scan(&p.ID, &p.Title, &p.Description, &p.Banner, &p.Location, &p.Price, &p.CreatedAt, &p.UserID, &p.Username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return p, ErrNoRecord
		}
		return p, err
	}
	return p, nil
}

func (s *PropertyStorage) Save(userID, listingID int64) error {
	query := `
      insert into favorites (user_id, listing_id)
      values ($1, $2)
    `
	args := []any{userID, listingID}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok && pqErr.Code.Name() == "unique_violation" {
			return ErrDuplicateSave
		}
		return err
	}
	return nil
}

func (s *PropertyStorage) Unsave(userID, listingID int64) error {
	query := `
      delete from favorites
      where user_id = $1 and listing_id = $2
    `
	args := []any{userID, listingID}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := s.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	return nil
}
