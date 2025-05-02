package models

import "time"

type Payment struct {
	ID          int   `gorm:"primaryKey;autoIncrement"`
	UserID      int64 `gorm:"not null"`
	User        User
	Amount      float64   `gorm:"not null"`
	Date        time.Time `gorm:"autoCreateTime"`
	Payload     string    `gorm:"size:512;not null"`
	IsCompleted bool      `gorm:"default:false"`
}
