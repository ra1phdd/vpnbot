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

type Servers struct{}

func NewServers() *Servers {
	return &Servers{}
}

func (s *Servers) GetAll() (servers []models.Server, err error) {
	cacheKey := "servers:all"
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &servers)
		return servers, err
	}

	rows, err := db.Conn.Queryx(`SELECT * FROM servers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var data models.Server
		if err := rows.StructScan(&data); err != nil {
			return nil, err
		}

		servers = append(servers, data)
	}

	jsonData, err := json.Marshal(servers)
	if err != nil {
		return nil, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 15*time.Minute).Err()
	if err != nil {
		return nil, err
	}

	return servers, nil
}

func (s *Servers) GetById(id int) (server models.Server, err error) {
	cacheKey := fmt.Sprintf("servers:id:%d", id)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return models.Server{}, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &server)
		return server, err
	}

	err = db.Conn.QueryRowx(`SELECT * FROM servers WHERE id = $1`, id).StructScan(&server)
	if err != nil {
		return models.Server{}, err
	}

	jsonData, err := json.Marshal(server)
	if err != nil {
		return models.Server{}, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 15*time.Minute).Err()
	if err != nil {
		return models.Server{}, err
	}

	return server, nil
}

func (s *Servers) GetByIP(ip string) (server models.Server, err error) {
	cacheKey := fmt.Sprintf("servers:ip:%s", ip)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return models.Server{}, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &server)
		return server, err
	}

	err = db.Conn.QueryRowx(`SELECT * FROM servers WHERE ip = $1`, ip).StructScan(&server)
	if err != nil {
		return models.Server{}, err
	}

	jsonData, err := json.Marshal(server)
	if err != nil {
		return models.Server{}, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 15*time.Minute).Err()
	if err != nil {
		return models.Server{}, err
	}

	return server, nil
}

func (s *Servers) GetByCC(countryId int) (servers []models.Server, int error) {
	cacheKey := fmt.Sprintf("servers:country_id:%d", countryId)
	cacheValue, err := cache.Rdb.Get(cache.Ctx, cacheKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	} else if cacheValue != "" {
		err = json.Unmarshal([]byte(cacheValue), &servers)
		return servers, err
	}

	rows, err := db.Conn.Queryx(`SELECT * FROM servers WHERE country_id = $1`, countryId)
	if err != nil {
		return []models.Server{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var data models.Server
		if err := rows.StructScan(&data); err != nil {
			return nil, err
		}

		servers = append(servers, data)
	}

	jsonData, err := json.Marshal(servers)
	if err != nil {
		return nil, err
	}
	err = cache.Rdb.Set(cache.Ctx, cacheKey, jsonData, 15*time.Minute).Err()
	if err != nil {
		return nil, err
	}

	return servers, nil
}

func (s *Servers) Delete(id int) error {
	_, err := db.Conn.Exec(`DELETE FROM servers WHERE id = $1`, id)
	return err
}

func (s *Servers) Add(server models.Server) error {
	_, err := db.Conn.Exec(`INSERT INTO servers (ip, port, country_id, channel_speed, private_key, public_key, dest, server_names, short_ids) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, server.IP, server.Port, server.CountryID, server.PrivateKey, server.PublicKey, server.Dest, server.ServerNames, server.ShortIDs)
	return err
}
