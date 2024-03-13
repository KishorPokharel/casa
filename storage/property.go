package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

var ErrDuplicateSave = errors.New("listing is already saved")

// type Property struct {
// 	ID          int64
// 	UserID      int64
// 	Banner      string
// 	Location    string
// 	Title       string
// 	Description string
// 	Price       int64
// 	Username    string
// 	CreatedAt   time.Time
// 	Rank        float64
// 	Saved       bool
// }

type Property struct {
	ID           int64
	UserID       int64
	Title        string
	Description  string
	PropertyType string
	Latitude     float64
	Longitude    float64
	Banner       string
	Location     string
	Price        int64
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Other Fields
	Pictures []string
	Username string
	Rank     float64
	Saved    bool
}

type PropertyFilter struct {
	Location     string
	PropertyType string
	MinPrice     int64
	MaxPrice     int64
}

type PropertyStorage struct {
	DB *sql.DB
}

func (s *PropertyStorage) ExistsWithID(id int64) (bool, error) {
	query := "select exists( select true from listings where id = $1 )"

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var exists bool
	err := s.DB.QueryRowContext(ctx, query, id).Scan(&exists)
	return exists, err
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
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}

	queryInsertListing := `
      insert into listings
        (title, user_id, description, banner, location, property_type, latitude, longitude, price)
      values
        ($1, $2, $3, $4, $5, $6, $7, $8, $9)
      returning id
    `
	args := []any{
		property.Title,
		property.UserID,
		property.Description,
		property.Banner,
		property.Location,
		property.PropertyType,
		property.Latitude, property.Longitude,
		property.Price,
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var listingID int64
	row := tx.QueryRowContext(ctx, queryInsertListing, args...)
	if err := row.Scan(&listingID); err != nil {
		tx.Rollback()
		return err
	}

	if len(property.Pictures) > 0 {
		queryInsertPictures := "insert into pictures (listing_id, url) values"
		for i := range len(property.Pictures) {
			if i != 0 {
				queryInsertPictures += ","
			}
			queryInsertPictures += fmt.Sprintf(" ($1, $%d)", i+2)
		}
		stmt, err := tx.Prepare(queryInsertPictures)
		if err != nil {
			tx.Rollback()
			return err
		}
		args := []any{listingID}
		for _, picture := range property.Pictures {
			args = append(args, picture)
		}
		_, err = stmt.ExecContext(ctx, args...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// func (s *PropertyStorage) Insert(property Property) error {
// 	query := `
//       insert into listings
//       (title, user_id, description, banner, location, property_type, price)
//       values ($1, $2, $3, $4, $5, $6, $7)
//     `
// 	args := []any{property.Title, property.UserID, property.Description, property.Banner, property.Location, "land", property.Price}

// 	ctx, cancel := context.WithTimeout(context.Background(), timeout)
// 	defer cancel()

// 	_, err := s.DB.ExecContext(ctx, query, args...)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

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

func (s *PropertyStorage) Search2(filter PropertyFilter) ([]Property, error) {
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
        and
            (listings.property_type = $2 or $2='')
        and
            ((listings.price >= $3 and listings.price <= $4) or ($3 = 0 and $4 = 0))
        order by rank desc
    `
	args := []any{filter.Location, filter.PropertyType, filter.MinPrice, filter.MaxPrice}

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

func (s *PropertyStorage) GetByID(id int64) (Property, error) {
	query := `
        select
            listings.id, listings.title, listings.description, listings.banner, listings.location, listings.property_type,
            listings.price, listings.latitude, listings.longitude, listings.created_at, users.id, users.username
        from
            listings
        join
            users on listings.user_id = users.id
        where listings.id = $1
    `

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	row := s.DB.QueryRowContext(ctx, query, id)
	p := Property{
		Pictures: []string{},
	}
	err := row.Scan(
		&p.ID,
		&p.Title,
		&p.Description,
		&p.Banner,
		&p.Location,
		&p.PropertyType,
		&p.Price,
		&p.Latitude,
		&p.Longitude,
		&p.CreatedAt,
		&p.UserID,
		&p.Username,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return p, ErrNoRecord
		}
		return p, err
	}

	queryPictures := `
        select url from pictures
        where listing_id = $1 and deleted_at is null
    `

	rows, err := s.DB.QueryContext(ctx, queryPictures, id)
	if err != nil {
		return p, err
	}
	for rows.Next() {
		var imageURL string
		if err := rows.Scan(&imageURL); err != nil {
			return p, err
		}
		p.Pictures = append(p.Pictures, imageURL)

	}
	return p, nil
}

// func (s *PropertyStorage) Get(id int64) (Property, error) {
// 	query := `
//         select
//             listings.id, listings.title, listings.description, listings.banner, listings.location,
//             listings.price, listings.created_at, users.id, users.username
//         from
//             listings
//         join
//             users on listings.user_id = users.id
//         where listings.id = $1
//     `

// 	ctx, cancel := context.WithTimeout(context.Background(), timeout)
// 	defer cancel()

// 	row := s.DB.QueryRowContext(ctx, query, id)
// 	p := Property{}
// 	err := row.Scan(&p.ID, &p.Title, &p.Description, &p.Banner, &p.Location, &p.Price, &p.CreatedAt, &p.UserID, &p.Username)
// 	if err != nil {
// 		if errors.Is(err, sql.ErrNoRows) {
// 			return p, ErrNoRecord
// 		}
// 		return p, err
// 	}
// 	return p, nil
// }

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

func (s *PropertyStorage) IsSaved(userID, listingID int64) (bool, error) {
	query := `
	  select exists(
        select true from favorites where user_id = $1 and listing_id = $2
      )
    `

	var exists bool
	args := []any{userID, listingID}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := s.DB.QueryRowContext(ctx, query, args...).Scan(&exists)
	return exists, err
}

func (s *PropertyStorage) GetSavedListings(userID int64) ([]Property, error) {
	query := `
        select 
            listings.id, listings.title, listings.description, listings.banner, listings.location,
            listings.price, listings.created_at, users.id, users.username
        from
            listings
        join
            favorites on favorites.listing_id = listings.id
        join
            users on listings.user_id = users.id
        where
            favorites.user_id = $1
        order by favorites.created_at desc
    `

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	rows, err := s.DB.QueryContext(ctx, query, userID)
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

func (s *PropertyStorage) GetAllForUser(userID int64) ([]Property, error) {
	query := `
        select 
            listings.id, listings.title, listings.description, listings.banner, listings.location,
            listings.price, listings.created_at, users.id, users.username
        from
            listings
        join
            users on listings.user_id = users.id
        where
            listings.user_id = $1
        order by listings.created_at desc
    `

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	rows, err := s.DB.QueryContext(ctx, query, userID)
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

func (s *PropertyStorage) GetAllLocations() ([]string, error) {
	query := `
        select distinct location from listings
    `

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	locations := []string{}
	for rows.Next() {
		var location string
		err := rows.Scan(&location)
		if err != nil {
			return nil, err
		}
		locations = append(locations, location)
	}
	return locations, nil
}

func (s *PropertyStorage) GetMinMaxPrice() (int64, int64, error) {
	query := `select min(price), max(price) from listings`

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	row := s.DB.QueryRowContext(ctx, query)
	var min, max int64
	if err := row.Scan(&min, &max); err != nil {
		return min, max, err
	}

	return min, max, nil
}

func (s *PropertyStorage) Update(property Property) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}

	queryUpdateListing := `
      update listings
      set
        title = $1, 
        price = $2,
        description = $3,
        banner = $4,
        location = $5,
        property_type = $6,
        latitude = $7,
        longitude = $8
      where
        id = $9 and user_id = $10
    `

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []any{
		property.Title,
		property.Price,
		property.Description,
		property.Banner,
		property.Location,
		property.PropertyType,
		property.Latitude,
		property.Longitude,
		property.ID,
		property.UserID,
	}
	_, err = tx.ExecContext(ctx, queryUpdateListing, args...)
	if err != nil {
		tx.Rollback()
		return err
	}

	if len(property.Pictures) > 0 {
		queryInsertPictures := "insert into pictures (listing_id, url) values"
		for i := range len(property.Pictures) {
			if i != 0 {
				queryInsertPictures += ","
			}
			queryInsertPictures += fmt.Sprintf(" ($1, $%d)", i+2)
		}
		stmt, err := tx.Prepare(queryInsertPictures)
		if err != nil {
			tx.Rollback()
			return err
		}
		args := []any{property.ID}
		for _, picture := range property.Pictures {
			args = append(args, picture)
		}
		_, err = stmt.ExecContext(ctx, args...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
