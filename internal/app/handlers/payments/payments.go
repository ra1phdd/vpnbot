package payments

import (
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v3"
	"nsvpn/internal/app/models"
	"time"
)

type Payments interface {
	Add(data models.Payment) (int, error)
}
type Subscriptions interface {
	Add(data models.Subscription) (int, error)
	IsActive(userId int64) (bool, error)
}

type Endpoint struct {
	Bot           *telebot.Bot
	Payments      Payments
	Subscriptions Subscriptions
}

func (e *Endpoint) PaymentHandler(c telebot.Context) error {
	isActive, err := e.Subscriptions.IsActive(c.Sender().ID)
	if err != nil {
		return err
	}

	if isActive {
		return c.Send("Ошибка! У вас уже есть действующая подписка. Дождитесь её окончания")
	}

	u, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	invoice := CreateInvoice("XTR", fmt.Sprint(u))

	err = c.Send(&invoice)
	if err != nil {
		return err
	}

	return nil
}

func (e *Endpoint) PreCheckoutHandler(c telebot.Context) error {
	sub := models.Subscription{
		UserID:  c.Sender().ID,
		EndDate: time.Now().UTC().AddDate(0, 0, 30),
	}

	subId, err := e.Subscriptions.Add(sub)
	if err != nil {
		return err
	}

	payment := models.Payment{
		UserID:         c.Sender().ID,
		Amount:         c.PreCheckoutQuery().Total,
		Currency:       c.PreCheckoutQuery().Currency,
		SubscriptionID: int64(subId),
		Uuid:           c.PreCheckoutQuery().Payload,
	}

	_, err = e.Payments.Add(payment)
	if err != nil {
		return err
	}

	//err = e.Bot.Accept(c.PreCheckoutQuery())
	//if err != nil {
	//	return err
	//}

	return c.Send(fmt.Sprintf("Платёж успешно завершен! Номер платежа: %s", c.PreCheckoutQuery().Payload))
}

func CreateInvoice(currency string, uuid string) telebot.Invoice {
	var amount int
	switch currency {
	case "XTR":
		amount = 100
	default:
		amount = 1
	}

	invoice := telebot.Invoice{
		Title:       "Оплата подписки",
		Description: "Подписка NSVPN на 1 месяц.",
		Payload:     uuid,
		Currency:    currency,
		Prices: []telebot.Price{
			{
				Label:  "Оплата подписки",
				Amount: amount,
			},
		},
	}

	return invoice
}
