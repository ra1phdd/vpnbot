package models

import "time"

type RequestStatus struct {
	ID         int    `db:"id"`
	StatusName string `db:"status_name"`
}

type SupportRequest struct {
	ID          int       `db:"id"`
	UserID      int64     `db:"user_id"`
	StatusID    int       `db:"status_id"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
	Description string    `db:"description"`
}

type Message struct {
	ID        int       `db:"id"`
	RequestID int       `db:"request_id"`
	UserID    int64     `db:"user_id"`
	Message   string    `db:"message"`
	CreatedAt time.Time `db:"created_at"`
}
