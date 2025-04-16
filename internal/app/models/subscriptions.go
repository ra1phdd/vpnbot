package models

import "time"

type Subscription struct {
	ID        int `gorm:"primaryKey;autoIncrement"`
	UserID    int64
	User      User
	StartDate time.Time `gorm:"autoCreateTime"`
	EndDate   *time.Time
	IsActive  bool `gorm:"default:false"`
}
