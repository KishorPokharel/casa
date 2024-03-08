package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Room struct {
	ID uuid.UUID
	// Other
	User User
}

type Message struct {
	UserID    int64
	RoomID    uuid.UUID
	Text      string
	CreatedAt time.Time
}

type MessageStorage struct {
	DB *sql.DB
}

func (s *MessageStorage) GetAllRoomsForUser(userID int64) ([]Room, error) {
	query := `
        SELECT r.id AS room_id, u.username AS username
        FROM rooms r
        INNER JOIN users_rooms ur ON r.id = ur.room_id
        INNER JOIN users u ON ur.user_id = u.id
        WHERE ur.room_id IN (
            SELECT room_id
            FROM users_rooms
            WHERE user_id = $1
        ) AND ur.user_id != $1
    `
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	rows, err := s.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}

	rooms := []Room{}
	for rows.Next() {
		var room Room
		if err := rows.Scan(&room.ID, &room.User.Username); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

func (s *MessageStorage) CheckRoomExists(id1, id2 int64) (uuid.UUID, error) {
	query := `
        select ura.room_id from users_rooms as ura
        join users_rooms as urb
        on ura.room_id = urb.room_id
        where ura.user_id = $1 and urb.user_id = $2
    `
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	row := s.DB.QueryRowContext(ctx, query, id1, id2)
	var roomID uuid.UUID
	err := row.Scan(&roomID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return roomID, ErrNoRecord
		}
		return roomID, err
	}
	return roomID, nil
}

func (s *MessageStorage) NewRoom(userID, ownerID int64) (uuid.UUID, error) {
	roomID := uuid.New()

	tx, err := s.DB.Begin()
	if err != nil {
		return roomID, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	queryCreateRoom := "insert into rooms (id) values ($1)"
	_, err = tx.ExecContext(ctx, queryCreateRoom, roomID)
	if err != nil {
		tx.Rollback()
		return roomID, err
	}

	queryInsertUser := "insert into users_rooms(user_id, room_id) values ($1, $2)"
	_, err = tx.ExecContext(ctx, queryInsertUser, userID, roomID)
	if err != nil {
		tx.Rollback()
		return roomID, err
	}

	_, err = tx.ExecContext(ctx, queryInsertUser, ownerID, roomID)
	if err != nil {
		tx.Rollback()
		return roomID, err
	}
	if err := tx.Commit(); err != nil {
		return roomID, err
	}

	return roomID, nil
}
