package handlers

import (
	"fmt"
	"go.uber.org/atomic"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/api"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"strings"
	"sync"
)

type Servers struct {
	log           *logger.Logger
	bot           *telebot.Bot
	kh            *Keys
	ss            *services.Servers
	cs            *services.Country
	countriesBtns *services.Buttons
	api           *api.API
}

func NewServers(log *logger.Logger, bot *telebot.Bot, ss *services.Servers, kh *Keys, cs *services.Country, api *api.API) *Servers {
	countries, err := cs.GetAll()
	if err != nil {
		log.Error("Failed to get countries from db", err)
		return nil
	}

	buttons, layout := cs.ProcessButtons(countries)
	countriesBtns := services.NewButtons(buttons, layout, "inline")

	s := &Servers{
		log:           log,
		bot:           bot,
		kh:            kh,
		ss:            ss,
		cs:            cs,
		countriesBtns: countriesBtns,
		api:           api,
	}

	countriesMapBtns := countriesBtns.GetBtns()
	for _, btn := range countriesMapBtns {
		bot.Handle(btn, s.CountryHandler)
	}

	return s
}

func (s *Servers) ListCountriesHandler(c telebot.Context) error {
	if err := validateSubscription(c); err != nil {
		return err
	}

	return c.Send("✈️ Список доступных стран", s.countriesBtns.AddBtns())
}

func (s *Servers) CountryHandler(c telebot.Context) error {
	btns := getReplyButtons(c)
	if err := validateSubscription(c); err != nil {
		return err
	}

	country, err := s.cs.Get(c.Callback().Unique)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	return s.InfoHandler(c, country)
}

func (s *Servers) InfoHandler(c telebot.Context, country *models.Country) error {
	btns := getReplyButtons(c)
	if err := validateSubscription(c); err != nil {
		return err
	}

	servers, err := s.ss.GetAllByCountryID(country.ID)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	loadInfo := s.calculateServerLoad(servers)
	msg := s.buildMessage(country, loadInfo)

	getKeyBtn := services.NewButtons([]models.ButtonOption{{
		Value:   "get_key_" + strings.ToLower(country.Code),
		Display: "📥 Получить ключ",
	}}, []int{1}, "inline")
	for _, btn := range getKeyBtn.GetBtns() {
		s.bot.Handle(btn, func(c telebot.Context) error {
			return s.kh.GetKeyHandler(c, country)
		})
	}

	return c.Edit(msg, getKeyBtn.AddBtns())
}

type LoadInfo struct {
	TotalLoad  float64
	Inactive   int64
	TotalCount int
}

func (s *Servers) calculateServerLoad(servers []*models.Server) LoadInfo {
	var (
		wg       sync.WaitGroup
		sumLoad  atomic.Float64
		inActive atomic.Int64
	)

	for _, serv := range servers {
		wg.Add(1)
		go func(server *models.Server) {
			defer wg.Done()
			if load, err := s.api.GetLoadRequest(server); err != nil {
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

func (s *Servers) buildMessage(country *models.Country, info LoadInfo) string {
	loadMsg := s.getLoadStatusMessage(info)
	return fmt.Sprintf("%s %s\n🎛 Нагрузка на сервер: %s\n\n",
		country.Emoji,
		country.Code,
		loadMsg,
	)
}

func (s *Servers) getLoadStatusMessage(info LoadInfo) string {
	if int64(info.TotalCount) == info.Inactive {
		return "не отвечает 🔴"
	}

	activeCount := float64(info.TotalCount) - float64(info.Inactive)
	avgLoad := info.TotalLoad / activeCount

	switch {
	case avgLoad <= 0.3:
		return "низкая 🟢"
	case avgLoad <= 0.7:
		return "средняя 🌕"
	case avgLoad <= 0.95:
		return "высокая 🟠"
	default:
		return "критическая 🔴"
	}
}
