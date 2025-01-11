package services

import (
	"gopkg.in/telebot.v3"
	"nsvpn/internal/app/models"
)

type Buttons struct {
	bot *telebot.Bot
}

func NewButtons(bot *telebot.Bot) *Buttons {
	return &Buttons{
		bot: bot,
	}
}

func (b *Buttons) createButtonRows(menu *telebot.ReplyMarkup, buttons []models.ButtonOption, layout []int) (rows []telebot.Row) {
	btns := make(map[string]telebot.Btn, len(buttons))
	for _, item := range buttons {
		btns[item.Value] = menu.Text(item.Display)
	}

	start := 0
	for _, count := range layout {
		if start+count > len(buttons) {
			break
		}
		row := make(telebot.Row, 0, count)
		for i := start; i < start+count; i++ {
			row = append(row, btns[buttons[i].Value])
		}
		rows = append(rows, row)
		start += count
	}
	return
}

func (b *Buttons) ReplyWithButtons(menu *telebot.ReplyMarkup, buttons []models.ButtonOption, layout []int) {
	rows := b.createButtonRows(menu, buttons, layout)
	menu.Reply(rows...)
}

func (b *Buttons) InlineWithButtons(menu *telebot.ReplyMarkup, buttons []models.ButtonOption, layout []int) {
	rows := b.createButtonRows(menu, buttons, layout)
	menu.Inline(rows...)
}
