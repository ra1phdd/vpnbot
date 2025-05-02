package models

import "time"

type Subscription struct {
	ID        int `gorm:"primaryKey;autoIncrement"`
	UserID    int64
	User      User
	StartDate time.Time `gorm:"autoCreateTime"`
	EndDate   time.Time
	IsActive  bool `gorm:"default:false"`
}

type SubscriptionPlan struct {
	ID                int               `gorm:"primaryKey;autoIncrement"`
	Name              string            `gorm:"size:64;unique;not null"`
	DurationDays      int               `gorm:"not null"`
	SubscriptionPrice SubscriptionPrice `gorm:"not null"`
}

type SubscriptionPrice struct {
	ID                 int     `gorm:"primaryKey;autoIncrement"`
	SubscriptionPlanID int     `gorm:"unique;not null"`
	Price              float64 `gorm:"not null"`
}
