package models

type Country struct {
	ID          uint   `gorm:"primaryKey;autoIncrement"`
	Code        string `gorm:"size:2;unique;not null"`
	Emoji       string `gorm:"size:64;not null"`
	NameRU      string `gorm:"size:64;not null"`
	NameEN      string `gorm:"size:64;not null"`
	Domain      string `gorm:"size:64;not null"`
	PrivateKey  string `gorm:"size:512;unique;not null"`
	PublicKey   string `gorm:"size:512;unique;not null"`
	Flow        string `gorm:"size:255"`
	Dest        string `gorm:"size:255;not null"`
	ServerNames string `gorm:"size:255;not null"`
	ShortIDs    string `gorm:"size:255;not null"`
}
