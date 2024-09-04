package servers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/pkg/db"
	"strconv"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s Service) Get() ([]models.Server, error) {
	var servers []models.Server

	rows, err := db.Conn.Query(`SELECT * FROM servers`)
	if err != nil {
		return []models.Server{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var data models.Server
		err = rows.Scan(&data.ID, &data.IP, &data.CountryCode, &data.Cookie)
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

func (s Service) GetByCC(countryCode string) (models.Server, error) {
	var data models.Server

	rows, err := db.Conn.Query(`SELECT * FROM servers WHERE country_code = $1`, countryCode)
	if err != nil {
		return models.Server{}, err
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&data.ID, &data.IP, &data.CountryCode, &data.Cookie)
		if err != nil {
			return models.Server{}, err
		}
	}
	if data.ID == 0 {
		return models.Server{}, constants.ErrServerNotFound
	}

	return data, nil
}

func (s Service) Delete(countryCode string) error {
	rows, err := db.Conn.Queryx(`DELETE FROM servers WHERE country_code = $1`, countryCode)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

func (s Service) Add(server models.Server) error {
	rows, err := db.Conn.Queryx(`INSERT INTO servers (id, ip, country_code, cookie) VALUES ($1, $2, $3, $4)`, server.ID, server.IP, server.CountryCode, server.Cookie)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

func (s Service) AddClient(id int64) error {
	servers, err := s.Get()
	if err != nil {
		return err
	}

	hash := sha256.Sum256([]byte(strconv.FormatInt(id, 10)))
	u := hex.EncodeToString(hash[:])

	settings := models.Settings{
		Clients: []models.Client{
			{
				ID:         fmt.Sprint(u),
				Flow:       "",
				Email:      fmt.Sprint(id),
				LimitIp:    0,
				TotalGB:    0,
				ExpiryTime: -2592000000,
				Enable:     true,
				TgId:       "",
				SubId:      "",
				Reset:      0,
			},
		},
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	data := url.Values{}
	data.Set("id", "1")
	data.Set("settings", string(settingsJSON))

	for _, server := range servers {
		if len(server.CountryCode) == 2 {
			continue
		}

		req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:12053/panel/inbound/addClient", server.IP), bytes.NewBufferString(data.Encode()))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json, text/plain, */*")
		req.Header.Set("Cookie", fmt.Sprintf("3x-ui=%s; lang=ru-RU", server.Cookie))
		req.Header.Set("Origin", fmt.Sprintf("http://%s:12053", server.IP))
		req.Header.Set("Referer", fmt.Sprintf("http://%s:12053/panel/inbounds", server.IP))
		req.Header.Set("X-Requested-With", "XMLHttpRequest")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Чтение ответа
		if resp.StatusCode != 200 {
			return fmt.Errorf("ответ от сервера: %s", resp.Status)
		}
	}

	return nil
}
