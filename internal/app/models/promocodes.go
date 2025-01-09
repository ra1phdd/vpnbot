package models

import "time"

type PromoCode struct {
	ID                 int    `db:"id"`
	Code               string `db:"code"`
	Discount           int    `db:"discount"`
	TotalActivations   *int   `db:"total_activations"`
	CurrentActivations int    `db:"current_activations"`
	OnlyNewUsers       bool   `db:"only_new_users"`
	IsActive           bool   `db:"is_active"`
}

type PromoCodeActivation struct {
	ID          int       `db:"id"`
	PromoCodeID int       `db:"promocode_id"`
	UserID      int64     `db:"user_id"`
	ActivatedAt time.Time `db:"activated_at"`
}
