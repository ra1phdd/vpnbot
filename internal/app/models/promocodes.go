package models

import "time"

type Promocode struct {
	ID                 uint   `gorm:"primaryKey;autoIncrement"`
	Code               string `gorm:"size:15;unique;not null"`
	Discount           int    `gorm:"not null"`
	TotalActivations   uint
	CurrentActivations uint      `gorm:"default:0"`
	OnlyNewUsers       bool      `gorm:"default:false"`
	IsActive           bool      `gorm:"default:true"`
	UserID             *int64    `gorm:"default:null"`
	User               User      `gorm:"foreignKey:UserID;references:ID"`
	StartAt            time.Time `gorm:"autoCreateTime"`
	EndAt              *time.Time
}

type PromocodeActivations struct {
	ID          uint `gorm:"primaryKey"`
	PromocodeID uint `gorm:"not null"`
	Promocode   Promocode
	UserID      int64 `gorm:"not null"`
	User        User
	ActivatedAt time.Time `gorm:"autoCreateTime"`
}
