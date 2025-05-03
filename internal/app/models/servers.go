package models

type Server struct {
	ID           uint    `gorm:"primaryKey;autoIncrement"`
	IP           string  `gorm:"size:15;unique;not null"`
	CountryID    uint    `gorm:"not null"`
	Country      Country `gorm:"foreignKey:CountryID;references:ID"`
	ChannelSpeed uint64  `gorm:"not null"`
	Port         uint    `gorm:"not null"`
}
