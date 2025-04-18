package handlers

import (
	"fmt"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/api"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"sync"
)

type Servers struct {
	log           *logger.Logger
	bot           *telebot.Bot
	kh            *Keys
	ss            *services.Servers
	cs            *services.Country
	countriesBtns *services.Buttons
	serversBtns   *services.Buttons
}

func NewServers(log *logger.Logger, bot *telebot.Bot, ss *services.Servers, kh *Keys, cs *services.Country) *Servers {
	countries, err := cs.GetAll()
	if err != nil {
		log.Error("Failed to get countries from db", err)
		return nil
	}

	buttons, layout := cs.ProcessButtons(countries)
	countriesBtns := services.NewButtons(buttons, layout, "reply")

	s := &Servers{
		log:           log,
		bot:           bot,
		kh:            kh,
		ss:            ss,
		cs:            cs,
		countriesBtns: countriesBtns,
	}

	countriesMapBtns := countriesBtns.GetBtns()
	for _, btn := range countriesMapBtns {
		bot.Handle(btn, s.CountryHandler)
	}

	return s
}

func (s *Servers) ListCountriesHandler(c telebot.Context) error {
	return c.Send("Список доступных стран", s.countriesBtns.AddBtns())
}

func (s *Servers) CountryHandler(c telebot.Context) error {
	btns := s.countriesBtns.GetBtns()
	for i, btn := range btns {
		if btn.Text == c.Text() {
			country := models.Country{
				CountryCode: i,
				CountryName: btn.Text,
			}

			return s.InfoHandler(c, country)
		}
	}

	return fmt.Errorf("country not found")
}

func (s *Servers) InfoHandler(c telebot.Context, country models.Country) error {
	servers, err := s.ss.GetByCC(country.CountryCode)
	if err != nil {
		return c.Send("Упс! Что-то сломалось. Повторите попытку позже")
	}

	var msg string
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]string, len(servers))

	for i, serv := range servers {
		wg.Add(1)
		go func(idx int, serv models.Server) {
			defer wg.Done()

			var loadMsg string
			sa := api.NewServer(serv)
			load, err := sa.GetLoadRequest()
			if err != nil {
				loadMsg = "не отвечает 🔴"
			} else {
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

			mu.Lock()
			results[idx] = fmt.Sprintf("%s-%d\n🚀 IP-адрес: %s\n🎛 Нагрузка на сервер: %s\n\n", country.CountryName, idx+1, serv.IP, loadMsg)
			mu.Unlock()
		}(i, serv)
	}
	wg.Wait()

	for _, line := range results {
		msg += line
	}
	msg += "Получить ключ для сервера:"

	buttons, layout := s.ss.ProcessButtons(country, servers)
	s.serversBtns = services.NewButtons(buttons, layout, "inline")

	serversMapBtns := s.serversBtns.GetBtns()
	i := 0
	for _, btn := range serversMapBtns {
		server := servers[i]
		s.bot.Handle(btn, func(c telebot.Context) error {
			return s.ServerHandler(c, server, btn.Text)
		})
		i++
	}

	return c.Send(msg, s.serversBtns.AddBtns())
}

func (s *Servers) ServerHandler(c telebot.Context, server models.Server, countryName string) error {
	btns := s.serversBtns.GetBtns()
	for _, btn := range btns {
		if btn.Text == countryName {
			return s.kh.GetKeyHandler(c, server, countryName, btn.Unique)
		}
	}

	return fmt.Errorf("server not found")
}
