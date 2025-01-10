package servers

import (
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/db"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s Service) GetAll() ([]models.Server, error) {
	var servers []models.Server

	rows, err := db.Conn.Query(`SELECT * FROM servers`)
	if err != nil {
		return []models.Server{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var data models.Server
		err = rows.Scan(&data.ID, &data.IP, &data.CountryID, &data.ChannelSpeed, &data.PrivateKey, &data.PublicKey, &data.Dest, &data.ServerNames, &data.ShortIDs)
		if err != nil {
			return []models.Server{}, err
		}

		servers = append(servers, data)
	}
	if len(servers) == 0 {
		return []models.Server{}, constants.ErrServerNotFound
	}

	return servers, nil
}

func (s Service) GetById(id int) (models.Server, error) {
	var data models.Server

	rows, err := db.Conn.Query(`SELECT * FROM servers WHERE id = $1`, id)
	if err != nil {
		return models.Server{}, err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&data.ID, &data.IP, &data.CountryID, &data.ChannelSpeed, &data.PrivateKey, &data.PublicKey, &data.Dest, &data.ServerNames, &data.ShortIDs)
		if err != nil {
			return models.Server{}, err
		}
	}
	if data.ID == 0 {
		return models.Server{}, constants.ErrServerNotFound
	}

	return data, nil
}

func (s Service) GetByIp(ip int) (models.Server, error) {
	var data models.Server

	rows, err := db.Conn.Query(`SELECT * FROM servers WHERE ip = $1`, ip)
	if err != nil {
		return models.Server{}, err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&data.ID, &data.IP, &data.CountryID, &data.ChannelSpeed, &data.PrivateKey, &data.PublicKey, &data.Dest, &data.ServerNames, &data.ShortIDs)
		if err != nil {
			return models.Server{}, err
		}
	}
	if data.ID == 0 {
		return models.Server{}, constants.ErrServerNotFound
	}

	return data, nil
}

func (s Service) GetByCC(countryCode string) ([]models.Server, error) {
	var servers []models.Server

	rows, err := db.Conn.Query(`SELECT * FROM servers WHERE country_code = $1`, countryCode)
	if err != nil {
		return []models.Server{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var data models.Server
		err = rows.Scan(&data.ID, &data.IP, &data.CountryID, &data.ChannelSpeed, &data.PrivateKey, &data.PublicKey, &data.Dest, &data.ServerNames, &data.ShortIDs)
		if err != nil {
			return []models.Server{}, err
		}

		servers = append(servers, data)
	}
	if len(servers) == 0 {
		return []models.Server{}, constants.ErrServerNotFound
	}

	return servers, nil
}

func (s Service) Delete(id int) error {
	rows, err := db.Conn.Queryx(`DELETE FROM servers WHERE id = $1`, id)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

func (s Service) Add(server models.Server) error {
	rows, err := db.Conn.Queryx(`INSERT INTO servers (id, ip, country_id, channel_speed, private_key, public_key, dest, server_names, short_ids) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, server.ID, server.IP, server.CountryID, server.PrivateKey, server.PublicKey, server.Dest, server.ServerNames, server.ShortIDs)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}
