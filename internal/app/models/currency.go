package models

type Currency struct {
	ID           int    `db:"id"`
	CurrencyCode string `db:"currency_code"`
}
