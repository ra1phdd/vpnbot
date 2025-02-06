package models

type Key struct {
	ID           int    `db:"id"`
	UserID       int64  `db:"user_id"`
	ServerID     int    `db:"server_id"`
	UUID         string `db:"uuid"`
	SpeedLimit   int64  `db:"speed_limit"`
	TrafficLimit int64  `db:"traffic_limit"`
	TrafficUsed  int64  `db:"traffic_used"`
	IsActive     bool   `db:"is_active"`
}
