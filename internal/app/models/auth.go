package models

import "time"

type Token struct {
	ID        int       `db:"id"`
	UserID    int64     `db:"user_id"`
	Token     string    `db:"token"`
	CreatedAt time.Time `db:"created_at"`
	ExpiresAt time.Time `db:"expires_at"`
}
