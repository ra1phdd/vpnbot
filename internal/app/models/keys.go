package models

type Key struct {
	ID           int   `gorm:"primaryKey;autoIncrement"`
	UserID       int64 `gorm:"not null"`
	User         User
	ServerID     int `gorm:"not null"`
	Server       Server
	UUID         string `gorm:"size:512;not null"`
	SpeedLimit   int
	TrafficLimit int64
	TrafficUsed  int64
	IsActive     bool `gorm:"default:true"`
}
