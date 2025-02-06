package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
)

type Servers struct {
	bot                        *telebot.Bot
	kh                         *Keys
	ss                         *services.Servers
	cs                         *services.Country
	countriesBtns, serversBtns *services.Buttons
}

func NewServers(bot *telebot.Bot, kh *Keys, cs *services.Country, ss *services.Servers) *Servers {
	countries, err := cs.GetCountries()
	if err != nil {
		logger.Error("Failed to get countries from DB", zap.Error(err))
		return nil
	}

	buttons, layout := ss.ProcessCountries(countries)
	countriesBtns := services.NewButtons(buttons, layout, "reply")

	s := &Servers{
		bot:           bot,
		kh:            kh,
		ss:            ss,
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
		return err
	}

	var msg string
	for i, serv := range servers {
		authKey := sha256.Sum256([]byte(fmt.Sprintf("%s%s", serv.PublicKey, serv.PrivateKey)))
		load, err := s.ss.GetLoadRequest(serv.IP, serv.Port, hex.EncodeToString(authKey[:]))
		if err != nil {
			return err
		}

		var loadMsg string
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

		msg += fmt.Sprintf("%s-%d\n🚀 IP-адрес: %s\n🎛 Нагрузка на сервер: %s\n\n", country.CountryName, i+1, serv.IP, loadMsg)
	}
	msg += "Получить ключ для сервера:"

	buttons, layout := s.ss.ProcessServers(country, servers)
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
