package models

type Key struct {
	ID           uint  `gorm:"primaryKey;autoIncrement"`
	UserID       int64 `gorm:"not null"`
	User         User
	CountryID    uint `gorm:"not null"`
	Country      Country
	UUID         string `gorm:"size:512;not null"`
	SpeedLimit   uint64
	TrafficLimit uint64
	TrafficUsed  uint64
	IsActive     bool `gorm:"default:true"`
}
