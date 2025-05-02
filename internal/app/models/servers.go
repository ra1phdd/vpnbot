package models

type Server struct {
	ID           int     `gorm:"primaryKey;autoIncrement"`
	IP           string  `gorm:"size:15;unique;not null"`
	CountryID    int     `gorm:"not null"`
	Country      Country `gorm:"foreignKey:CountryID;references:ID"`
	ChannelSpeed int     `gorm:"not null"`
	Port         int     `gorm:"not null"`
}
