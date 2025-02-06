package models

type Country struct {
	ID          int    `db:"id"`
	CountryCode string `db:"country_code"`
	CountryName string `db:"country_name"`
}
