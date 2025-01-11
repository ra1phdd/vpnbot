package repository

import (
	"database/sql"
	"errors"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/db"
)

type Servers struct{}

func NewServers() *Servers {
	return &Servers{}
}

func (s *Servers) GetAll() ([]models.Server, error) {
	var servers []models.Server

	rows, err := db.Conn.Queryx(`SELECT * FROM servers`)
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
	if len(servers) == 0 {
		return []models.Server{}, constants.ErrServerNotFound
	}

	return servers, nil
}

func (s *Servers) GetById(id int) (models.Server, error) {
	var data models.Server

	err := db.Conn.QueryRowx(`SELECT * FROM servers WHERE id = $1`, id).StructScan(&data)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Server{}, constants.ErrServerNotFound
		}
		return models.Server{}, err
	}

	return data, nil
}

func (s *Servers) GetByIP(ip int) (models.Server, error) {
	var data models.Server

	err := db.Conn.QueryRowx(`SELECT * FROM servers WHERE ip = $1`, ip).StructScan(&data)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Server{}, constants.ErrServerNotFound
		}
		return models.Server{}, err
	}

	return data, nil
}

func (s *Servers) GetByCC(countryCode string) ([]models.Server, error) {
	var servers []models.Server

	rows, err := db.Conn.Queryx(`SELECT * FROM servers WHERE country_code = $1`, countryCode)
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
	if len(servers) == 0 {
		return []models.Server{}, constants.ErrServerNotFound
	}

	return servers, nil
}

func (s *Servers) Delete(id int) error {
	_, err := db.Conn.Exec(`DELETE FROM servers WHERE id = $1`, id)
	return err
}

func (s *Servers) Add(server models.Server) error {
	_, err := db.Conn.Exec(`INSERT INTO servers (id, ip, country_id, channel_speed, private_key, public_key, dest, server_names, short_ids) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, server.ID, server.IP, server.CountryID, server.PrivateKey, server.PublicKey, server.Dest, server.ServerNames, server.ShortIDs)
	return err
}
