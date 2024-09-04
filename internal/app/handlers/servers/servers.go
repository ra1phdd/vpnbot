package servers

import (
	"gopkg.in/telebot.v3"
	"nsvpn/internal/app/models"
	"strconv"
)

type Server interface {
	Add(server models.Server) error
	AddClient(id int64) error
}

type Endpoint struct {
	Server Server
}

func (e *Endpoint) AddServerHandler(c telebot.Context) error {
	if c.Sender().ID != 1230045591 {
		return nil
	}

	args := c.Args()

	// /addserv ip code cookie
	if len(args) != 3 {
		return c.Send("Неверный формат команды. Пожалуйста, используйте: /addserv ip code.")
	}

	server := models.Server{
		IP:          args[0],
		CountryCode: args[1],
		Cookie:      args[2],
	}

	err := e.Server.Add(server)
	if err != nil {
		return err
	}

	return c.Send("Успешно!")
}

func (e *Endpoint) AddClientHandler(c telebot.Context) error {
	if c.Sender().ID != 1230045591 {
		return nil
	}

	args := c.Args()

	// /addcl id
	if len(args) != 1 {
		return c.Send("Неверный формат команды. Пожалуйста, используйте: /addcl id.")
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return c.Send("Неверный формат суммы. Пожалуйста, используйте правильный формат, например: /addcl id")
	}

	err = e.Server.AddClient(id)
	if err != nil {
		return err
	}

	return c.Send("Успешно!")
}
