package services

import (
	"fmt"
	"go.uber.org/atomic"
	"nsvpn/internal/app/api"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/pkg/logger"
	"strings"
	"sync"
)

type Servers struct {
	log *logger.Logger
	sr  *repository.Servers
	api *api.API
}

func NewServers(log *logger.Logger, sr *repository.Servers, api *api.API) *Servers {
	return &Servers{
		log: log,
		sr:  sr,
		api: api,
	}
}

func (ss *Servers) GetAll() (servers []*models.Server, err error) {
	return ss.sr.GetAll()
}

func (ss *Servers) GetAllByCountryID(countryID uint) (servers []*models.Server, err error) {
	if countryID == 0 {
		return nil, constants.ErrEmptyFields
	}

	return ss.sr.GetAllByCountryID(countryID)
}

func (ss *Servers) Get(id uint) (server *models.Server, err error) {
	if id == 0 {
		return nil, constants.ErrEmptyFields
	}

	return ss.sr.Get(id)
}

func (ss *Servers) Add(server *models.Server) error {
	if server.IP == "" || server.CountryID == 0 || server.ChannelSpeed == 0 || server.Port == 0 {
		return constants.ErrEmptyFields
	}

	return ss.sr.Add(server)
}

func (ss *Servers) Update(id uint, newServer *models.Server) error {
	if id == 0 || newServer.IP == "" || newServer.CountryID == 0 || newServer.ChannelSpeed == 0 || newServer.Port == 0 {
		return constants.ErrEmptyFields
	}

	return ss.sr.Update(id, newServer)
}

func (ss *Servers) Delete(id uint) error {
	if id == 0 {
		return constants.ErrEmptyFields
	}

	return ss.sr.Delete(id)
}

func (ss *Servers) ProcessButtons(servers []models.Server) ([]models.ButtonOption, []int) {
	listServers := make([]models.ButtonOption, 0, len(servers))

	for i, server := range servers {
		listServers = append(listServers, models.ButtonOption{
			Value:   fmt.Sprintf("server_%s-%d", strings.ToLower(server.Country.Code), i+1),
			Display: fmt.Sprintf("%s %s-%d", server.Country.Emoji, server.Country.Code, i+1),
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

type LoadInfo struct {
	TotalLoad  float64
	Inactive   int64
	TotalCount int
}

func (ss *Servers) CalculateServerLoad(servers []*models.Server) LoadInfo {
	var (
		wg       sync.WaitGroup
		sumLoad  atomic.Float64
		inActive atomic.Int64
	)

	for _, serv := range servers {
		wg.Add(1)
		go func(server *models.Server) {
			defer wg.Done()
			if load, err := ss.api.GetLoadRequest(server); err != nil {
				inActive.Add(1)
			} else {
				sumLoad.Add(load)
			}
		}(serv)
	}
	wg.Wait()

	return LoadInfo{
		TotalLoad:  sumLoad.Load(),
		Inactive:   inActive.Load(),
		TotalCount: len(servers),
	}
}

func (ss *Servers) BuildMessage(country *models.Country, info LoadInfo) string {
	loadMsg := "критическая 🔴"
	avgLoad := info.TotalLoad / (float64(info.TotalCount) - float64(info.Inactive))

	switch {
	case avgLoad <= 0.3:
		loadMsg = "низкая 🟢"
	case avgLoad <= 0.7:
		loadMsg = "средняя 🌕"
	case avgLoad <= 0.95:
		loadMsg = "высокая 🟠"
	case int64(info.TotalCount) == info.Inactive:
		loadMsg = "не отвечает 🔴"
	}

	return fmt.Sprintf("%s %s\n🎛 Нагрузка на сервер: %s\n\n",
		country.Emoji,
		country.Code,
		loadMsg,
	)
}
