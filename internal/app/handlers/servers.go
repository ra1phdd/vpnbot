package handlers

import (
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/services"
	"nsvpn/internal/app/state"
	"nsvpn/pkg/logger"
	"strconv"
)

type Servers struct {
	log           *logger.Logger
	bot           *telebot.Bot
	kh            *Keys
	ss            *services.Servers
	subs          *services.Subscriptions
	cs            *services.Country
	hashCountries string
	countriesBtns *services.Buttons
}

func NewServers(log *logger.Logger, bot *telebot.Bot, ss *services.Servers, subs *services.Subscriptions, kh *Keys, cs *services.Country) *Servers {
	return &Servers{
		log:  log,
		bot:  bot,
		kh:   kh,
		ss:   ss,
		subs: subs,
		cs:   cs,
	}
}

func (s *Servers) RegisterHandlers() {
	countries, err := s.cs.GetAll()
	if err != nil {
		s.log.Error("Failed to get countries from db", err)
		return
	}

	if hash, err := s.cs.GetHash(countries); err != nil || s.hashCountries == hash {
		return
	} else {
		s.hashCountries = hash
	}

	buttons, layout := s.cs.ProcessButtons(countries)
	s.countriesBtns = services.NewButtons(buttons, layout, "inline")

	for _, btn := range s.countriesBtns.GetBtns() {
		s.bot.Handle(btn, s.CountryHandler)
	}
}

func (s *Servers) ListCountriesHandler(c telebot.Context) error {
	if err := validateSubscription(c, s.subs); err != nil {
		return err
	}

	s.RegisterHandlers()
	return c.Send("✈️ Список доступных стран", s.countriesBtns.AddBtns())
}

func (s *Servers) CountryHandler(c telebot.Context) error {
	btns := getReplyButtons(c)
	if err := validateSubscription(c, s.subs); err != nil {
		return err
	}

	country, err := s.cs.Get(c.Callback().Unique)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	servers, err := s.ss.GetAllByCountryID(country.ID)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	s.kh.KeysState.Set(strconv.FormatInt(c.Sender().ID, 10), state.KeysState{
		Country: country,
	})

	msg := s.ss.BuildMessage(country, s.ss.CalculateServerLoad(servers))
	return c.Edit(msg, s.kh.GetBtns.AddBtns())
}
