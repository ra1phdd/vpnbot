package models

type Currency struct {
	ID           int     `gorm:"primaryKey;autoIncrement"`
	Code         string  `gorm:"size:3;unique;not null"`
	Symbol       string  `gorm:"size:3;unique;not null"`
	Name         string  `gorm:"size:64;not null"`
	ExchangeRate float64 `gorm:"not null;default:1"` // Курс к базовой валюте (например USD)
	IsBase       bool    `gorm:"default:false"`      // Является ли базовой валютой
}
