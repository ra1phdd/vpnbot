package models

type User struct {
	ID        int64  `gorm:"primaryKey"`
	Username  string `gorm:"size:32"`
	Firstname string `gorm:"size:64"`
	Lastname  string `gorm:"size:64"`
	PartnerID int64  `gorm:""`
	Balance   int    `gorm:""`
	IsAdmin   bool   `gorm:"default:false"`
	IsSign    bool   `gorm:"default:false"`
}
