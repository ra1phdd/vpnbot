package services

import (
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/models"
)

type Buttons struct {
	typeKeyboard string
	rows         []telebot.Row
	btns         map[string]*telebot.Btn
	menu         *telebot.ReplyMarkup
}

func NewButtons(buttons []models.ButtonOption, layout []int, typeKeyboard string) *Buttons {
	var menu *telebot.ReplyMarkup
	switch typeKeyboard {
	case "reply":
		menu = &telebot.ReplyMarkup{ResizeKeyboard: true, IsPersistent: true}
	case "inline":
		menu = &telebot.ReplyMarkup{}
	}
	rows, btns := createButtonRows(menu, buttons, layout, typeKeyboard)

	return &Buttons{
		typeKeyboard: typeKeyboard,
		rows:         rows,
		btns:         btns,
		menu:         menu,
	}
}

func createButtonRows(menu *telebot.ReplyMarkup, buttons []models.ButtonOption, layout []int, typeKeyboard string) ([]telebot.Row, map[string]*telebot.Btn) {
	btns := make(map[string]*telebot.Btn, len(buttons))

	switch typeKeyboard {
	case "reply":
		for _, item := range buttons {
			btn := menu.Text(item.Display)
			btns[item.Value] = &btn
		}
	case "inline":
		for _, item := range buttons {
			btn := menu.Data(item.Display, item.Value)
			btns[item.Value] = &btn
		}
	}

	var rows []telebot.Row
	start := 0
	for _, count := range layout {
		if start+count > len(buttons) {
			break
		}
		row := make(telebot.Row, 0, count)
		for i := start; i < start+count; i++ {
			row = append(row, *btns[buttons[i].Value])
		}
		rows = append(rows, row)
		start += count
	}
	return rows, btns
}

func (b *Buttons) GetBtns() map[string]*telebot.Btn {
	return b.btns
}

func (b *Buttons) AddBtns() *telebot.ReplyMarkup {
	switch b.typeKeyboard {
	case "reply":
		b.menu.Reply(b.rows...)
	case "inline":
		b.menu.Inline(b.rows...)
	}

	return b.menu
}
