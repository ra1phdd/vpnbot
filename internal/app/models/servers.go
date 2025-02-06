package models

import "time"

type Server struct {
	ID           int    `db:"id"`
	IP           string `db:"ip"`
	Port         int    `db:"port"`
	CountryID    int    `db:"country_id"`
	ChannelSpeed int    `db:"channel_speed"`
	PrivateKey   string `db:"private_key"`
	PublicKey    string `db:"public_key"`
	Dest         string `db:"dest"`
	ServerNames  string `db:"server_names"`
	ShortIDs     string `db:"short_ids"`
}

type ServerStatistics struct {
	ID                   int       `db:"id"`
	ServerID             int       `db:"server_id"`
	Timestamp            time.Time `db:"timestamp"`
	ActiveConnections    int       `db:"active_connections"`
	AverageDownloadSpeed int64     `db:"average_download_speed"`
	AverageUploadSpeed   int64     `db:"average_upload_speed"`
}
