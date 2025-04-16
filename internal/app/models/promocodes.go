package models

import "time"

type Promocode struct {
	ID                 int    `gorm:"primaryKey;autoIncrement"`
	Code               string `gorm:"size:15;unique;not null"`
	Discount           int    `gorm:"not null"`
	TotalActivations   *int
	CurrentActivations int  `gorm:"default:0"`
	OnlyNewUsers       bool `gorm:"default:false"`
	IsActive           bool `gorm:"default:true"`
}

type PromocodeActivations struct {
	ID          int `gorm:"primaryKey"`
	PromocodeID int `gorm:"not null"`
	Promocode   Promocode
	UserID      int64 `gorm:"not null"`
	User        User
	ActivatedAt time.Time `gorm:"autoCreateTime"`
}
