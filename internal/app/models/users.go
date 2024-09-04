package models

import "time"

type User struct {
	ID        int64
	Username  string
	Firstname string
	Lastname  string
}

type Payment struct {
	ID             int64
	UserID         int64
	Amount         int
	Currency       string
	Date           time.Time
	SubscriptionID int64
	Uuid           string
}

type Subscription struct {
	ID        int64
	UserID    int64
	StartDate time.Time
	EndDate   time.Time
}

type Server struct {
	ID          int64
	IP          string
	CountryCode string
	Cookie      string
}

type Client struct {
	ID         string `json:"id"`
	Flow       string `json:"flow"`
	Email      string `json:"email"`
	LimitIp    int    `json:"limitIp"`
	TotalGB    int    `json:"totalGB"`
	ExpiryTime int64  `json:"expiryTime"`
	Enable     bool   `json:"enable"`
	TgId       string `json:"tgId"`
	SubId      string `json:"subId"`
	Reset      int    `json:"reset"`
}
type Settings struct {
	Clients []Client `json:"clients"`
}
