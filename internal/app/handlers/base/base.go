package base

import (
	"gopkg.in/telebot.v3"
)

type Base interface {
}

type Endpoint struct {
	Base Base
}

func (e *Endpoint) HelpHandler(c telebot.Context) error {
	err := c.Send("🚀 Базовые команды\n/help - Посмотреть справку о командах\n")
	if err != nil {
		return err
	}
	return nil
}
