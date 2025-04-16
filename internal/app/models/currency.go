package models

type Currency struct {
	ID           int    `gorm:"primaryKey;autoIncrement"`
	CurrencyCode string `gorm:"size:3;unique;not null"`
	CurrencyName string `gorm:"size:64;not null"`
}
