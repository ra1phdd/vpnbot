package handlers

import (
	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
)

type Servers struct {
	ss            *services.Servers
	countriesBtns *services.Buttons
}

func NewServers(ss *services.Servers) *Servers {
	countries, err := ss.GetCountries()
	if err != nil {
		logger.Error("Failed to get countries from DB", zap.Error(err))
		return nil
	}

	buttons, layout := ss.ProcessCountries(countries)
	btns := services.NewButtons(buttons, layout, "reply")

	return &Servers{
		ss:            ss,
		countriesBtns: btns,
	}
}

func (s *Servers) ListCountries(c telebot.Context) error {
	return c.Send("Список доступных стран", s.countriesBtns.AddBtns())
}

//func (e *Endpoint) GetServerHandler(c telebot.Context) error {
//	if c.Sender().ID != 1230045591 {
//		return nil
//	}
//
//	args := c.Args()
//	var servers []models.Server
//
//	// /get code
//	if len(args) == 1 {
//		item, err := e.Server.GetByCC(args[0])
//		if err != nil {
//			return err
//		}
//
//		servers = append(servers, item)
//	} else {
//		var err error
//		servers, err = e.Server.Get()
//		if err != nil {
//			return err
//		}
//	}
//
//	msg := "Список серверов:\n"
//	for _, server := range servers {
//		msg += fmt.Sprintf("- %s (%s)", server.IP, server.CountryCode)
//	}
//
//	return c.Send(msg)
//}
//
//func (e *Endpoint) AddServerHandler(c telebot.Context) error {
//	if c.Sender().ID != 1230045591 {
//		return nil
//	}
//
//	args := c.Args()
//
//	// /addserv ip code cookie
//	if len(args) != 3 {
//		return c.Send("Неверный формат команды. Пожалуйста, используйте: /addserv ip code.")
//	}
//
//	server := models.Server{
//		IP:          args[0],
//		CountryCode: args[1],
//		Cookie:      args[2],
//	}
//
//	err := e.Server.Add(server)
//	if err != nil {
//		return err
//	}
//
//	return c.Send("Успешно!")
//}
//
//func (e *Endpoint) DeleteServerHandler(c telebot.Context) error {
//	if c.Sender().ID != 1230045591 {
//		return nil
//	}
//
//	args := c.Args()
//
//	// /delserv code
//	if len(args) != 1 {
//		return c.Send("Неверный формат команды. Пожалуйста, используйте: /delserv code.")
//	}
//
//	err := e.Server.Delete(args[0])
//	if err != nil {
//		return err
//	}
//
//	return c.Send("Успешно!")
//}
//
//func (e *Endpoint) AddClientHandler(c telebot.Context) error {
//	if c.Sender().ID != 1230045591 {
//		return nil
//	}
//
//	args := c.Args()
//
//	// /addcl id
//	if len(args) != 1 {
//		return c.Send("Неверный формат команды. Пожалуйста, используйте: /addcl id.")
//	}
//
//	id, err := strconv.ParseInt(args[0], 10, 64)
//	if err != nil {
//		return c.Send("Неверный формат суммы. Пожалуйста, используйте правильный формат, например: /addcl id")
//	}
//
//	err = e.Server.AddClient(id)
//	if err != nil {
//		return err
//	}
//
//	return c.Send("Успешно!")
//}
