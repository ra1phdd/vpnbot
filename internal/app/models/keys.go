package models

type Key struct {
	ID           uint  `gorm:"primaryKey;autoIncrement"`
	UserID       int64 `gorm:"not null;uniqueIndex:idx_user_country"`
	User         User
	CountryID    uint `gorm:"not null;uniqueIndex:idx_user_country"`
	Country      Country
	UUID         string `gorm:"size:512;not null"`
	SpeedLimit   uint64
	TrafficLimit uint64
	TrafficUsed  uint64
	IsActive     bool `gorm:"default:true"`
}
