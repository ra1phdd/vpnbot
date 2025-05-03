package models

import "time"

type Payment struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	UserID      int64     `gorm:"not null"`
	Amount      float64   `gorm:"not null"`
	Type        string    `gorm:"size:10;not null"` // "income" или "expense"
	Date        time.Time `gorm:"autoCreateTime"`
	Payload     string    `gorm:"size:512;not null"`
	Note        string    `gorm:"size:500"`
	IsCompleted bool      `gorm:"default:false"`
}
