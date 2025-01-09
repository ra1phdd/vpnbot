package models

import "time"

type Payment struct {
	ID             int       `db:"id"`
	UserID         int64     `db:"user_id"`
	Amount         float64   `db:"amount"`
	CurrencyID     int       `db:"currency_id"`
	Date           time.Time `db:"date"`
	SubscriptionID int       `db:"subscription_id"`
	Payload        string    `db:"payload"`
}
