package models

import "time"

type Subscription struct {
	ID        uint `gorm:"primaryKey;autoIncrement"`
	UserID    int64
	User      User
	StartDate time.Time `gorm:"autoCreateTime"`
	EndDate   time.Time
	IsActive  bool `gorm:"default:false"`
}

type SubscriptionPlan struct {
	ID                uint              `gorm:"primaryKey;autoIncrement"`
	Name              string            `gorm:"size:64;unique;not null"`
	DurationDays      uint              `gorm:"not null"`
	SubscriptionPrice SubscriptionPrice `gorm:"not null"`
}

type SubscriptionPrice struct {
	ID                 uint    `gorm:"primaryKey;autoIncrement"`
	SubscriptionPlanID uint    `gorm:"unique;not null"`
	Price              float64 `gorm:"not null"`
}
