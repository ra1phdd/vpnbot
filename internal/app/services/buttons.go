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

const (
	KeyboardTypeReply  = "reply"
	KeyboardTypeInline = "inline"
)

func NewButtons(buttons []models.ButtonOption, layout []int, typeKeyboard string) *Buttons {
	menu := &telebot.ReplyMarkup{}
	if typeKeyboard == KeyboardTypeReply {
		menu = &telebot.ReplyMarkup{ResizeKeyboard: true, IsPersistent: true}
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
	case KeyboardTypeReply:
		for _, item := range buttons {
			btn := menu.Text(item.Display)
			btns[item.Value] = &btn
		}
	case KeyboardTypeInline:
		for _, item := range buttons {
			var btn telebot.Btn
			if item.URL != "" {
				btn = menu.URL(item.Display, item.URL)
			} else {
				btn = menu.Data(item.Display, item.Value)
			}
			btns[item.Value] = &btn
		}
	}

	start := 0
	rows := make([]telebot.Row, 0, len(layout))
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

func (bs *Buttons) GetBtns() map[string]*telebot.Btn {
	return bs.btns
}

func (bs *Buttons) GetBtn(value string) *telebot.Btn {
	return bs.btns[value]
}

func (bs *Buttons) AddBtns() *telebot.ReplyMarkup {
	switch bs.typeKeyboard {
	case KeyboardTypeReply:
		bs.menu.Reply(bs.rows...)
	case KeyboardTypeInline:
		bs.menu.Inline(bs.rows...)
	}

	return bs.menu
}
