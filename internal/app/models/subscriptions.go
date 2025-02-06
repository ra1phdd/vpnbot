package models

import "time"

type Subscription struct {
	ID        int        `db:"id"`
	UserID    int64      `db:"user_id"`
	StartDate time.Time  `db:"start_date"`
	EndDate   *time.Time `db:"end_date"`
	IsActive  bool       `db:"is_active"`
}
