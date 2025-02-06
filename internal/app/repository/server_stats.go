package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/db"
	"time"
)

type ServerStats struct{}

func NewServerStats() *ServerStats {
	return &ServerStats{}
}

func (s *ServerStats) GetAllStats() (serversStats []models.ServerStatistics, err error) {
	cacheKey := "servers_stats:all"
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &serversStats)
		return serversStats, err
	}

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

	jsonData, err := json.Marshal(serversStats)
	if err != nil {
		return nil, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 15*time.Minute).Err()
	if err != nil {
		return nil, err
	}

	return serversStats, nil
}

func (s *ServerStats) GetLastStatsById(id int) (serverStats models.ServerStatistics, err error) {
	cacheKey := fmt.Sprintf("servers_stats:id:%d", id)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return models.ServerStatistics{}, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &serverStats)
		return serverStats, err
	}

	err = db.Conn.QueryRowx(`SELECT * FROM server_statistics WHERE id = $1 ORDER BY timestamp DESC LIMIT 1`, id).StructScan(&serverStats)
	if err != nil {
		return models.ServerStatistics{}, err
	}

	jsonData, err := json.Marshal(serverStats)
	if err != nil {
		return models.ServerStatistics{}, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 15*time.Minute).Err()
	if err != nil {
		return models.ServerStatistics{}, err
	}

	return serverStats, nil
}
