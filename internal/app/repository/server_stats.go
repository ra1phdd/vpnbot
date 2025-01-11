package repository

import (
	"database/sql"
	"errors"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/db"
)

type ServerStats struct{}

func NewServerStats() *ServerStats {
	return &ServerStats{}
}

func (s *ServerStats) GetAllStats() ([]models.ServerStatistics, error) {
	var serversStats []models.ServerStatistics

	rows, err := db.Conn.Queryx(`SELECT * FROM server_statistics`)
	if err != nil {
		return []models.ServerStatistics{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var data models.ServerStatistics
		if err := rows.StructScan(&data); err != nil {
			return nil, err
		}

		serversStats = append(serversStats, data)
	}
	if len(serversStats) == 0 {
		return []models.ServerStatistics{}, constants.ErrServerNotFound
	}

	return serversStats, nil
}

func (s *ServerStats) GetLastStatsById(id int) (models.ServerStatistics, error) {
	var data models.ServerStatistics

	err := db.Conn.QueryRowx(`SELECT * FROM server_statistics WHERE id = $1 ORDER BY timestamp DESC LIMIT 1`, id).StructScan(&data)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ServerStatistics{}, constants.ErrServerNotFound
		}
		return models.ServerStatistics{}, err
	}

	return data, nil
}
