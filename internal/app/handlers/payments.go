package handlers

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v4"
	"log/slog"
	"math"
	"net/http"
	"nsvpn/internal/app/config"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"strconv"
	"time"
)

type Payments struct {
	log    *logger.Logger
	bot    *telebot.Bot
	cfg    *config.Configuration
	cs     *services.Currency
	pcodes *services.Promocodes
	ps     *services.Payments
	us     *services.Users
	ph     *Promocodes
}

func NewPayments(log *logger.Logger, bot *telebot.Bot, cfg *config.Configuration, pcodes *services.Promocodes, ps *services.Payments, cs *services.Currency, us *services.Users, ph *Promocodes) *Payments {
	return &Payments{
		log:    log,
		bot:    bot,
		cfg:    cfg,
		cs:     cs,
		pcodes: pcodes,
		ps:     ps,
		us:     us,
		ph:     ph,
	}
}

func (p *Payments) RequestAmount(c telebot.Context, payload, note string) error {
	btns := getReplyButtons(c)
	baseCurrency, err := p.cs.GetIsBase()
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	if err := c.Send(fmt.Sprintf("💳 Введите сумму в %s:", baseCurrency.Code)); err != nil {
		p.log.Error("Failed to send message", err)
		return nil
	}

	resultChan := make(chan float64)
	defer close(resultChan)

	p.bot.Handle(telebot.OnText, func(c telebot.Context) error {
		amount, err := strconv.ParseFloat(c.Text(), 64)
		if err != nil || amount < 1 {
			return c.Send("❌ Некорректная сумма, попробуйте ещё раз", btns)
		}
		resultChan <- amount

		p.bot.Handle(telebot.OnText, func(c telebot.Context) error {
			return c.Send("🤔 Неизвестная команда. Используйте /help для получения списка команд", btns)
		})
		return nil
	})

	select {
	case amount := <-resultChan:
		return p.ChooseCurrencyHandler(c, math.Round(amount), payload, note, false)
	case <-time.After(5 * time.Minute):
		p.bot.Handle(telebot.OnText, func(c telebot.Context) error {
			return c.Send("🤔 Неизвестная команда. Используйте /help для получения списка команд", btns)
		})
		return c.Send("⌛ Время ввода суммы истекло", btns)
	}
}

func (p *Payments) ChooseCurrencyHandler(c telebot.Context, amount float64, payload, note string, isBuySub bool) error {
	btns := getReplyButtons(c)
	baseCurrency, err := p.cs.GetIsBase()
	if err != nil {
		p.log.Error("Failed to get base currency", err)
		return c.Send(constants.UserError, btns)
	}

	user, err := p.us.Get(c.Sender().ID)
	if err != nil {
		p.log.Error("Failed to get user", err)
		return c.Send(constants.UserError, btns)
	}

	chooseBtns := services.NewButtons([]models.ButtonOption{
		{Value: "pay_bankcard", Display: "💳 Банковская карта/СБП"},
		{Value: "pay_stars", Display: "⭐ Telegram Stars"},
		{Value: "pay_cryptocurrency", Display: "💎 Криптовалюта"},
	}, []int{1, 1, 1}, "inline")

	p.bot.Handle(chooseBtns.GetBtn("pay_bankcard"), func(c telebot.Context) error {
		return p.PaymentHandler(c, amount, "Оплата подписки на NSVPN", payload, note, "RUB")
	})
	p.bot.Handle(chooseBtns.GetBtn("pay_stars"), func(c telebot.Context) error {
		return p.PaymentHandler(c, amount, "Оплата подписки на NSVPN", payload, note, "XTR")
	})
	p.bot.Handle(chooseBtns.GetBtn("pay_cryptocurrency"), func(c telebot.Context) error {
		return p.PaymentHandler(c, amount, "Оплата подписки на NSVPN", payload, note, "BTC")
	})

	msg := fmt.Sprintf("💵 Сумма к оплате: %.f %s\n📦 Номер платежа: %s\n\nВыберите удобный для Вас способ оплаты:", amount, baseCurrency.Code, payload)
	if user.Balance > 0 && isBuySub {
		amount -= user.Balance
		msg = fmt.Sprintf("💰 Ваш текущий баланс: %.f %s\n💵 Сумма к оплате: %.f %s\n📦 Номер платежа: %s\n\nВыберите удобный для Вас способ оплаты:", user.Balance, baseCurrency.Code, amount, baseCurrency.Code, payload)
	}

	return c.Send(msg, chooseBtns.AddBtns())
}

func (p *Payments) PaymentHandler(c telebot.Context, amount float64, description, payload, note, currencyCode string) error {
	promocodeID, discount, err := p.ph.RequestPromocodeHandler(c)
	if err != nil {
		p.log.Error("Failed to request promocode handler", err)
	}

	btns := getReplyButtons(c)
	currency, err := p.cs.Get(currencyCode)
	if err != nil {
		return err
	}

	payment := &models.Payment{
		UserID:      c.Sender().ID,
		Amount:      amount,
		Type:        "income",
		Payload:     payload,
		Note:        note,
		IsCompleted: false,
	}

	invoiceAmount := amount
	if discount > 0 {
		discountAmount := amount * float64(discount) / 100
		invoiceAmount -= discountAmount
	}

	err = p.ps.Add(payment)
	if err != nil {
		p.log.Error("Failed add payment", err)
		return c.Send(constants.UserError, btns)
	}

	switch currencyCode {
	case "XTR":
		return p.TelegramPaymentHandler(c, invoiceAmount, description, payload, currency, promocodeID)
	case "RUB":
		return p.BankcardPaymentHandler(c, invoiceAmount, "kneshkreba@mail.ru", description, payload, promocodeID)
	case "BTC":
		return p.CryptoPaymentHandler(c, invoiceAmount, description, payload, promocodeID)
	default:
		p.log.Error("Unsupported currency", nil, slog.String("currency", currencyCode))
		return c.Send(constants.UserError, btns)
	}
}

func (p *Payments) BankcardPaymentHandler(c telebot.Context, amount float64, email, description, payload string, promocodeID uint) error {
	btns := getReplyButtons(c)

	paymentRequest := p.ps.CreateBankcardPayment(amount, email, description)
	jsonData, err := json.Marshal(paymentRequest)
	if err != nil {
		p.log.Error("Error marshaling JSON", err)
		return c.Send(constants.UserError, btns)
	}

	req, err := http.NewRequest("POST", p.cfg.YoukassaURL, bytes.NewBuffer(jsonData))
	if err != nil {
		p.log.Error("Error creating request", err)
		return c.Send(constants.UserError, btns)
	}

	req.SetBasicAuth(p.cfg.YoukassaID, p.cfg.YoukassaAPI)
	req.Header.Set("Idempotence-Key", payload)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		p.log.Error("Error making request", err)
		return c.Send(constants.UserError, btns)
	}
	defer resp.Body.Close()

	var response models.YoukassaResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		p.log.Error("Error decoding payment response", err)
		return c.Send(constants.UserError, btns)
	}

	menu := &telebot.ReplyMarkup{}
	row := telebot.Row{
		{
			Unique: "proceed_payment",
			Text:   "Перейти к оплате",
			URL:    response.Confirmation.ConfirmationURL,
		},
	}
	menu.Inline(row)

	go func() {
		err := p.ps.CheckBankcardPayment(response.ID)
		if err != nil {
			p.log.Error("Error checking payment response", err)
			return
		}

		err = p.SuccessfulPaymentHandler(c, amount, payload, promocodeID)
		if err != nil {
			p.log.Error("Error handling successful payment", err)
		}
	}()

	return c.Send("Пополнение баланса", menu)
}

func (p *Payments) CryptoPaymentHandler(c telebot.Context, amount float64, description, payload string, promocodeID uint) error {
	btns := getReplyButtons(c)

	paymentRequest := p.ps.CreateCryptoPayment(amount, description, payload)
	jsonData, err := json.Marshal(paymentRequest)
	if err != nil {
		p.log.Error("Error marshaling JSON", err)
		return c.Send(constants.UserError, btns)
	}

	b64 := base64.StdEncoding.EncodeToString(jsonData)
	hash := md5.Sum([]byte(b64 + p.cfg.HeleketAPI))

	req, err := http.NewRequest("POST", p.cfg.HeleketURL, bytes.NewBuffer(jsonData))
	if err != nil {
		p.log.Error("Error creating request", err)
		return c.Send(constants.UserError, btns)
	}
	req.Header.Set("merchant", p.cfg.HeleketID)
	req.Header.Set("sign", hex.EncodeToString(hash[:]))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		p.log.Error("Error making request", err)
		return c.Send(constants.UserError, btns)
	}
	defer resp.Body.Close()

	var response models.HeleketResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		p.log.Error("Error decoding payment response", err)
		return c.Send(constants.UserError, btns)
	}

	menu := &telebot.ReplyMarkup{}
	row := telebot.Row{
		{
			Unique: "proceed_payment",
			Text:   "Перейти к оплате",
			URL:    response.Result.URL,
		},
	}
	menu.Inline(row)

	go func() {
		err := p.ps.CheckCryptoPayment(response.Result.UUID, response.Result.OrderID)
		if err != nil {
			p.log.Error("Error checking payment response", err)
			return
		}

		err = p.SuccessfulPaymentHandler(c, amount, payload, promocodeID)
		if err != nil {
			p.log.Error("Error handling successful payment", err)
		}
	}()

	return c.Send("Пополнение баланса", menu)
}

func (p *Payments) TelegramPaymentHandler(c telebot.Context, amount float64, description, payload string, currency *models.Currency, promocodeID uint) error {
	p.bot.Handle(telebot.OnCheckout, func(c telebot.Context) error {
		return p.TelegramPreCheckoutHandler(c, amount)
	})
	p.bot.Handle(telebot.OnPayment, func(c telebot.Context) error {
		return p.SuccessfulPaymentHandler(c, amount, payload, promocodeID)
	})

	invoice := p.ps.CreateInvoice(math.Round(amount*currency.ExchangeRate), "Оплата", description, payload)
	return c.Send(&invoice)
}

func (p *Payments) TelegramPreCheckoutHandler(c telebot.Context, amount float64) error {
	p.log.Info("PreCheckout received", "user", c.Sender().ID, "payload", c.PreCheckoutQuery().Payload, "amount", c.PreCheckoutQuery().Total)

	btns := getReplyButtons(c)
	err := p.bot.Accept(c.PreCheckoutQuery())
	if err != nil {
		p.log.Error("Failed update isCompleted", err)

		err := p.ps.UpdateIsCompleted(c.Sender().ID, c.PreCheckoutQuery().Payload, false)
		if err != nil {
			p.log.Error("Failed update isCompleted", err)
		}

		err = p.us.DecrementBalance(c.Sender().ID, amount)
		if err != nil {
			p.log.Error("Failed update isCompleted", err)
		}

		return c.Send(constants.UserError, btns)
	}

	p.log.Info("PreCheckout accepted, waiting for payment confirmation")
	return nil
}

func (p *Payments) SuccessfulPaymentHandler(c telebot.Context, amount float64, payload string, promocodeID uint) error {
	btns := getReplyButtons(c)
	err := p.ps.UpdateIsCompleted(c.Sender().ID, payload, true)
	if err != nil {
		p.log.Error("Failed update isCompleted", err)
		return c.Send(constants.UserError, btns)
	}

	err = p.us.IncrementBalance(c.Sender().ID, amount)
	if err != nil {
		p.log.Error("Failed update isCompleted", err)
		return c.Send(constants.UserError, btns)
	}

	user, err := p.us.Get(c.Sender().ID)
	if err != nil {
		p.log.Error("Failed update isCompleted", err)
	} else if user.PartnerID != 0 && amount*0.15 >= 1 {
		err = p.ps.Add(&models.Payment{
			UserID:      user.PartnerID,
			Amount:      amount * 0.15,
			Type:        "income",
			Payload:     uuid.New().String(),
			Note:        "15% от пополнения баланса рефералом",
			IsCompleted: true,
		})
		if err != nil {
			p.log.Error("Failed add payment", err)
		}

		err = p.us.IncrementBalance(user.PartnerID, math.Round(amount*0.15))
		if err != nil {
			p.log.Error("Failed increment balance", err)
		}
	}

	if promocodeID != 0 {
		err = p.pcodes.Activations.Add(&models.PromocodeActivations{
			PromocodeID: promocodeID,
			UserID:      c.Sender().ID,
		})
		if err != nil {
			p.log.Error("Failed to activate promocode", err)
		}

		err = p.pcodes.IncrementActivationsByID(promocodeID)
		if err != nil {
			p.log.Error("Failed to increment activations promocode", err)
		}
	}

	return c.Send("✅ Платёж успешно завершен!", btns)
}

func (p *Payments) HistoryPaymentsHandler(c telebot.Context, currentPage int, isFirst bool) error {
	const pageSize = 15
	btns := getReplyButtons(c)

	totalCount, err := p.ps.GetPaymentsCount(c.Sender().ID)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}
	if totalCount == 0 {
		return c.Send("🧾 У вас пока нету оплаченных платежей", btns)
	}

	totalPages := totalCount / pageSize
	if totalCount%pageSize != 0 {
		totalPages++
	}

	if currentPage < 1 || currentPage > int(totalPages) {
		return nil
	}

	offset := (currentPage - 1) * pageSize
	payments, err := p.ps.GetAll(c.Sender().ID, offset, pageSize)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	baseCurrency, err := p.cs.GetIsBase()
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	msg := fmt.Sprintf("🧾 История платежей (страница %d из %d):\n", currentPage, totalPages)
	for i, payment := range payments {
		msg += fmt.Sprintf("%d) %s - %s - %.f %s\n", i+1+offset, payment.Date.Format("2006-01-02 15:04:05"), payment.Note, payment.Amount, baseCurrency.Code)
	}

	pgBtns := services.NewButtons([]models.ButtonOption{
		{
			Value:   "pagination_first",
			Display: "⏪",
		},
		{
			Value:   "pagination_prev",
			Display: "◀",
		},
		{
			Value:   "pagination_next",
			Display: "▶",
		},
		{
			Value:   "pagination_last",
			Display: "⏩",
		},
	}, []int{4}, "inline")
	p.bot.Handle(pgBtns.GetBtn("pagination_first"), func(c telebot.Context) error {
		return p.HistoryPaymentsHandler(c, 1, false)
	})
	p.bot.Handle(pgBtns.GetBtn("pagination_prev"), func(c telebot.Context) error {
		return p.HistoryPaymentsHandler(c, currentPage-1, false)
	})
	p.bot.Handle(pgBtns.GetBtn("pagination_next"), func(c telebot.Context) error {
		return p.HistoryPaymentsHandler(c, currentPage+1, false)
	})
	p.bot.Handle(pgBtns.GetBtn("pagination_last"), func(c telebot.Context) error {
		return p.HistoryPaymentsHandler(c, int(totalPages), false)
	})

	btns = &telebot.ReplyMarkup{}
	if totalPages > 1 {
		btns = pgBtns.AddBtns()
	}

	if isFirst {
		return c.Send(msg, btns)
	}
	return c.EditOrSend(msg, btns)
}
