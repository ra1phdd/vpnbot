package models

type User struct {
	ID        int64  `db:"id"`
	Username  string `db:"username"`
	Firstname string `db:"firstname"`
	Lastname  string `db:"lastname"`
	PartnerID *int   `db:"partner_id"`
}

type Partner struct {
	ID           int    `db:"id"`
	UserID       int64  `db:"user_id"`
	ReferralCode string `db:"referral_code"`
}
