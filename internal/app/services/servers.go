package services

import (
	"errors"
	"fmt"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
)

type Servers struct {
	log *logger.Logger
	sr  *repository.Servers
}

func NewServers(log *logger.Logger, sr *repository.Servers) *Servers {
	return &Servers{
		log: log,
		sr:  sr,
	}
}

func (ss *Servers) ProcessButtons(country models.Country, servers []models.Server) ([]models.ButtonOption, []int) {
	var listServers []models.ButtonOption

	for i := range servers {
		listServers = append(listServers, models.ButtonOption{
			Value:   fmt.Sprintf("%s-%d", country.CountryCode, i+1),
			Display: fmt.Sprintf("%s-%d", country.CountryName, i+1),
		})
	}

	var groups []int
	remaining := len(listServers)
	for remaining > 0 {
		if remaining >= 4 {
			groups = append(groups, 4)
			remaining -= 4
		} else {
			groups = append(groups, remaining)
			break
		}
	}

	return listServers, groups
}

func (ss *Servers) GetAll() ([]models.Server, error) {
	return ss.sr.GetAll()
}

func (ss *Servers) GetById(id int) (models.Server, error) {
	if id == 0 {
		return models.Server{}, errors.New("id is empty")
	}

	return ss.sr.GetById(id)
}

func (ss *Servers) GetByIP(ip string) (models.Server, error) {
	if ip == "" {
		return models.Server{}, errors.New("ip is empty")
	}

	return ss.sr.GetByIP(ip)
}

func (ss *Servers) GetByCC(countryCode string) ([]models.Server, error) {
	if countryCode == "" {
		return nil, errors.New("countryCode is empty")
	}

	return ss.sr.GetByСС(countryCode)
}

func (ss *Servers) Add(server models.Server) error {
	if server.IP == "" || server.CountryID == 0 || server.ChannelSpeed == 0 || server.PrivateKey == "" ||
		server.PublicKey == "" || server.Dest == "" || server.ServerNames == "" || server.ShortIDs == "" ||
		server.Port == 0 {
		return errors.New("server field is empty")
	}

	return ss.sr.Add(server)
}

func (ss *Servers) Delete(id int) error {
	if id == 0 {
		return errors.New("id is empty")
	}

	return ss.sr.Delete(id)
}
