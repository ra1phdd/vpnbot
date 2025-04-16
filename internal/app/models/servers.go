package models

type Server struct {
	ID           int    `gorm:"primaryKey;autoIncrement"`
	IP           string `gorm:"size:15;unique;not null"`
	CountryID    int    `gorm:"not null"`
	Country      Country
	ChannelSpeed int
	PrivateKey   string `gorm:"size:512;unique;not null"`
	PublicKey    string `gorm:"size:512;unique;not null"`
	Dest         string `gorm:"size:255;not null"`
	ServerNames  string `gorm:"size:255;not null"`
	ShortIDs     string `gorm:"size:255;not null"`
	Port         int
}
