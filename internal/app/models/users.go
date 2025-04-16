package models

type User struct {
	ID        int64    `gorm:"primaryKey"`
	Username  string   `gorm:"size:32"`
	Firstname string   `gorm:"size:64"`
	Lastname  string   `gorm:"size:64"`
	PartnerID *int     `gorm:"index"`
	Partner   *Partner `gorm:"foreignKey:PartnerID"`
	IsAdmin   bool     `gorm:"default:false"`
	IsSign    bool     `gorm:"default:false"`
}

type Partner struct {
	ID           int    `gorm:"primaryKey"`
	ReferralCode string `gorm:"size:32;unique;not null"`
}
