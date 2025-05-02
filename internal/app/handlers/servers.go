package handlers

import (
	"fmt"
	"go.uber.org/atomic"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/api"
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
	subs          *services.Subscriptions
	ss            *services.Servers
	cs            *services.Country
	countriesBtns *services.Buttons
	api           *api.API
}

func NewServers(log *logger.Logger, bot *telebot.Bot, subs *services.Subscriptions, ss *services.Servers, kh *Keys, cs *services.Country, api *api.API) *Servers {
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
		subs:          subs,
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
	if isActive, _ := s.subs.IsActive(c.Sender().ID, true); !isActive {
		return c.Send("У вас нет прав для выполнения данной команды")
	}

	return c.Send("Список доступных стран", s.countriesBtns.AddBtns())
}

func (s *Servers) CountryHandler(c telebot.Context) error {
	country, err := s.cs.Get(c.Callback().Unique)
	if err != nil {
		return err
	}

	return s.InfoHandler(c, country)
}

func (s *Servers) InfoHandler(c telebot.Context, country *models.Country) error {
	if isActive, _ := s.subs.IsActive(c.Sender().ID, true); !isActive {
		return c.Send("У вас нет прав для выполнения данной команды")
	}

	servers, err := s.ss.GetAllByCountryID(country.ID)
	if err != nil {
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	var wg sync.WaitGroup
	var sumLoad atomic.Float64
	var inActive atomic.Int64
	for _, serv := range servers {
		wg.Add(1)
		go func(server *models.Server) {
			defer wg.Done()

			load, err := s.api.GetLoadRequest(server)
			if err != nil {
				inActive.Add(1)
			}
			sumLoad.Add(load)
		}(serv)
	}
	wg.Wait()

	var loadMsg string
	if int64(len(servers)) == inActive.Load() {
		loadMsg = "не отвечает 🔴"
	} else {
		load := sumLoad.Load() / (float64(len(servers)) - float64(inActive.Load()))
		switch {
		case load <= 0.3:
			loadMsg = "низкая 🟢"
		case load > 0.3 && load <= 0.7:
			loadMsg = "средняя 🌕"
		case load > 0.7 && load <= 0.95:
			loadMsg = "высокая 🟠"
		case load > 0.95:
			loadMsg = "критическая 🔴"
		}
	}
	msg := fmt.Sprintf("%s %s\n🎛 Нагрузка на сервер: %s\n\n", country.Emoji, country.Code, loadMsg)

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
