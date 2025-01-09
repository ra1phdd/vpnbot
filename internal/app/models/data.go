package models

type Country struct {
	ID          int    `db:"id"`
	CountryCode string `db:"country_code"`
	CountryName string `db:"country_name"`
}

type Currency struct {
	ID           int    `db:"id"`
	CurrencyCode string `db:"currency_code"`
	CurrencyName string `db:"currency_name"`
}
