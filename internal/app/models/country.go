package models

type Country struct {
	ID          int    `gorm:"primaryKey;autoIncrement"`
	CountryCode string `gorm:"size:3;unique;not null"`
	CountryName string `gorm:"size:64;not null"`
}
