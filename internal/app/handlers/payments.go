package handlers

import (
	"fmt"
	"github.com/google/uuid"
	"gopkg.in/telebot.v4"
	"math"
	"nsvpn/internal/app/config"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/internal/app/state"
	"nsvpn/pkg/logger"
	"strconv"
	"time"
)

type Payments struct {
	log    *logger.Logger
	bot    *telebot.Bot
	cfg    *config.Configuration
	pcodes *services.Promocodes
	ps     *services.Payments
	us     *services.Users
	ph     *Promocodes

	chooseBtns    *services.Buttons
	historyPgBtns *services.Buttons

	PaymentsState   state.Storage[state.PaymentsState]
	paginationState state.Storage[state.PaginationState]
}

func NewPayments(log *logger.Logger, bot *telebot.Bot, cfg *config.Configuration,
	pcodes *services.Promocodes, ps *services.Payments, us *services.Users, ph *Promocodes) *Payments {
	return &Payments{
		log:    log,
		bot:    bot,
		cfg:    cfg,
		pcodes: pcodes,
		ps:     ps,
		us:     us,
		ph:     ph,

		chooseBtns: services.NewButtons([]models.ButtonOption{
			{Value: "pay_bankcard", Display: "💳 Банковская карта/СБП"},
			{Value: "pay_stars", Display: "⭐ Telegram Stars"},
			{Value: "pay_cryptocurrency", Display: "💎 Криптовалюта"},
		}, []int{1, 1, 1}, "inline"),
		historyPgBtns: services.NewButtons([]models.ButtonOption{
			{Value: "pagination_first", Display: "⏪"},
			{Value: "pagination_prev", Display: "◀"},
			{Value: "pagination_next", Display: "▶"},
			{Value: "pagination_last", Display: "⏩"},
		}, []int{4}, "inline"),

		PaymentsState:   state.NewMemoryStorage[state.PaymentsState](),
		paginationState: state.NewMemoryStorage[state.PaginationState](),
	}
}

func (p *Payments) RegisterRoutes() {
	paymentHandlers := map[string]func(c telebot.Context) error{
		"pay_bankcard":       p.BankcardPaymentHandler,
		"pay_stars":          p.TelegramPaymentHandler,
		"pay_cryptocurrency": p.CryptoPaymentHandler,
	}
	for value, handler := range paymentHandlers {
		p.bot.Handle(p.chooseBtns.GetBtn(value), p.CreatePaymentHandler(handler))
	}

	p.bot.Handle(telebot.OnCheckout, p.TelegramPreCheckoutHandler)
	p.bot.Handle(telebot.OnPayment, p.SuccessfulPaymentHandler)

	paginationHandlers := map[string]string{
		"pagination_first": "first",
		"pagination_prev":  "prev",
		"pagination_next":  "next",
		"pagination_last":  "last",
	}
	for value, handler := range paginationHandlers {
		p.bot.Handle(p.historyPgBtns.GetBtn(value), p.PaginationHandler(handler))
	}
}

func (p *Payments) CreatePaymentHandler(handler func(c telebot.Context) error) func(c telebot.Context) error {
	return func(c telebot.Context) error {
		btns := getReplyButtons(c)
		err := p.ph.RequestPromocodeHandler(c, p.PaymentsState)
		if err != nil {
			p.log.Error("Failed to request promocode handler", err)
		}

		ps, exists := p.PaymentsState.Get(strconv.FormatInt(c.Sender().ID, 10))
		if !exists {
			return c.Send(constants.UserError, btns)
		}

		payment := &models.Payment{
			UserID:      c.Sender().ID,
			Amount:      ps.Amount,
			Type:        "income",
			Payload:     ps.Payload,
			Note:        ps.Note,
			IsCompleted: false,
		}

		err = p.ps.Add(payment)
		if err != nil {
			p.log.Error("Failed add payment", err)
			return c.Send(constants.UserError, btns)
		}

		return handler(c)
	}
}

func (p *Payments) RequestAmount(c telebot.Context) error {
	btns := getReplyButtons(c)
	if err := c.Send("💳 Введите сумму в RUB:", btns); err != nil {
		p.log.Error("Failed to send message", err)
		p.PaymentsState.Delete(strconv.FormatInt(c.Sender().ID, 10))
		return nil
	}

	err := c.Respond()
	if err != nil {
		p.log.Error("Failed to send message", err)
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
		p.PaymentsState.Update(strconv.FormatInt(c.Sender().ID, 10), func(ps state.PaymentsState) state.PaymentsState {
			ps.Amount = amount
			return ps
		})

		return p.ChooseCurrencyHandler(c)
	case <-time.After(5 * time.Minute):
		p.bot.Handle(telebot.OnText, func(c telebot.Context) error {
			return c.Send("🤔 Неизвестная команда. Используйте /help для получения списка команд", btns)
		})

		p.PaymentsState.Delete(strconv.FormatInt(c.Sender().ID, 10))
		return c.Send("⌛ Время ввода суммы истекло", btns)
	}
}

func (p *Payments) ChooseCurrencyHandler(c telebot.Context) error {
	btns := getReplyButtons(c)
	user := getUser(c, p.us)
	if user == nil {
		p.log.Error("Failed to get user", nil)
		return c.Send(constants.UserError, btns)
	}

	ps, exists := p.PaymentsState.Get(strconv.FormatInt(c.Sender().ID, 10))
	if !exists {
		return c.Send(constants.UserError, btns)
	}

	var msg string
	if user.Balance > 0 && ps.IsBuySubscription {
		ps.Amount -= user.Balance
		msg = fmt.Sprintf("💰 Ваш текущий баланс: %.f RUB\n", user.Balance)
	}
	msg += fmt.Sprintf("💵 Сумма к оплате: %.f RUB\n📦 Номер платежа: %s\n\nВыберите удобный для Вас способ оплаты:", ps.Amount, ps.Payload)

	return c.Send(msg, p.chooseBtns.AddBtns())
}

func (p *Payments) BankcardPaymentHandler(c telebot.Context) error {
	defer func(c telebot.Context) {
		err := c.Respond()
		if err != nil {
			p.log.Error("Failed to send message", err)
		}
	}(c)

	btns := getReplyButtons(c)
	ps, exists := p.PaymentsState.Get(strconv.FormatInt(c.Sender().ID, 10))
	if !exists {
		return c.Send(constants.UserError, btns)
	}

	response, err := p.ps.CreateBankcardPayment(ps.Amount, "kneshkreba@mail.ru", ps.Description, ps.Payload)
	if err != nil {
		p.log.Error("Failed to create bankcard payment", err)
		return c.Send(constants.UserError, btns)
	}

	paymentBtns := services.NewButtons([]models.ButtonOption{
		{Value: "proceed_payment", Display: "Перейти к оплате", URL: response.Confirmation.ConfirmationURL},
		{Value: "tech_support", Display: "Техподдержка", URL: "https://t.me/nsvpn_support_bot"},
	}, []int{1, 1, 1}, "inline")

	go func(id string) {
		err := p.ps.CheckBankcardPayment(id)
		if err != nil {
			p.log.Error("Error checking payment response", err)
			return
		}

		err = p.SuccessfulPaymentHandler(c)
		if err != nil {
			p.log.Error("Error handling successful payment", err)
		}
	}(response.ID)

	return c.Send(p.ps.CreatePaymentMessage(ps.Amount, time.Now().Add(10*time.Minute), "Банковская карта/СБП", ps.Payload), paymentBtns.AddBtns())
}

func (p *Payments) CryptoPaymentHandler(c telebot.Context) error {
	defer func(c telebot.Context) {
		err := c.Respond()
		if err != nil {
			p.log.Error("Failed to send message", err)
		}
	}(c)

	btns := getReplyButtons(c)
	ps, exists := p.PaymentsState.Get(strconv.FormatInt(c.Sender().ID, 10))
	if !exists {
		return c.Send(constants.UserError, btns)
	}

	response, err := p.ps.CreateCryptoPayment(ps.Amount, ps.Description, ps.Payload)
	if err != nil {
		p.log.Error("Failed to create crypto payment", err)
		return c.Send(constants.UserError, btns)
	}

	paymentBtns := services.NewButtons([]models.ButtonOption{
		{Value: "proceed_payment", Display: "Перейти к оплате", URL: response.Result.URL},
		{Value: "tech_support", Display: "Техподдержка", URL: "https://t.me/nsvpn_support_bot"},
	}, []int{1, 1, 1}, "inline")

	go func() {
		err := p.ps.CheckCryptoPayment(response.Result.UUID, response.Result.OrderID)
		if err != nil {
			p.log.Error("Error checking payment response", err)
			return
		}

		err = p.SuccessfulPaymentHandler(c)
		if err != nil {
			p.log.Error("Error handling successful payment", err)
		}
	}()

	return c.Send(p.ps.CreatePaymentMessage(ps.Amount, time.Now().Add(10*time.Minute), "Криптовалюта", ps.Payload), paymentBtns.AddBtns())
}

func (p *Payments) TelegramPaymentHandler(c telebot.Context) error {
	defer func(c telebot.Context) {
		err := c.Respond()
		if err != nil {
			p.log.Error("Failed to send message", err)
		}
	}(c)

	btns := getReplyButtons(c)
	ps, exists := p.PaymentsState.Get(strconv.FormatInt(c.Sender().ID, 10))
	if !exists {
		return c.Send(constants.UserError, btns)
	}

	msg := p.ps.CreatePaymentMessage(ps.Amount, time.Now().Add(10*time.Minute), "Telegram Stars", ps.Payload)
	invoice := p.ps.CreateInvoice(math.Round(ps.Amount*0.6), "Пополнение баланса", msg, ps.Payload)
	return c.Send(&invoice)
}

func (p *Payments) TelegramPreCheckoutHandler(c telebot.Context) error {
	btns := getReplyButtons(c)
	err := p.bot.Accept(c.PreCheckoutQuery())
	if err != nil {
		p.log.Error("Failed update isCompleted", err)

		err := p.ps.UpdateIsCompleted(c.Sender().ID, c.PreCheckoutQuery().Payload, false)
		if err != nil {
			p.log.Error("Failed update isCompleted", err)
		}

		amount := float64(c.PreCheckoutQuery().Total) / 0.6
		if ps, exists := p.PaymentsState.Get(strconv.FormatInt(c.Sender().ID, 10)); exists {
			amount = ps.Amount
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

func (p *Payments) SuccessfulPaymentHandler(c telebot.Context) error {
	btns := getReplyButtons(c)
	ps, exists := p.PaymentsState.Get(strconv.FormatInt(c.Sender().ID, 10))
	if !exists {
		return c.Send(constants.UserError, btns)
	}

	err := p.ps.UpdateIsCompleted(c.Sender().ID, ps.Payload, true)
	if err != nil {
		p.log.Error("Failed update isCompleted", err)
	}

	err = p.us.IncrementBalance(c.Sender().ID, ps.Amount)
	if err != nil {
		p.log.Error("Failed update isCompleted", err)
	}

	user, err := p.us.Get(c.Sender().ID)
	if err != nil {
		p.log.Error("Failed update isCompleted", err)
	} else if user.PartnerID != 0 && ps.Amount*0.15 >= 1 {
		err = p.ps.Add(&models.Payment{
			UserID:      user.PartnerID,
			Amount:      ps.Amount * 0.15,
			Type:        "income",
			Payload:     uuid.New().String(),
			Note:        "15% от пополнения баланса рефералом",
			IsCompleted: true,
		})
		if err != nil {
			p.log.Error("Failed add payment", err)
		}

		err = p.us.IncrementBalance(user.PartnerID, math.Round(ps.Amount*0.15))
		if err != nil {
			p.log.Error("Failed increment balance", err)
		}
	}

	if ps.Promocode.ID != 0 {
		err = p.pcodes.Activations.Add(&models.PromocodeActivations{
			PromocodeID: ps.Promocode.ID,
			UserID:      c.Sender().ID,
		})
		if err != nil {
			p.log.Error("Failed to activate promocode", err)
		}

		err = p.pcodes.IncrementActivationsByID(ps.Promocode.ID)
		if err != nil {
			p.log.Error("Failed to increment activations promocode", err)
		}
	}

	p.PaymentsState.Delete(strconv.FormatInt(c.Sender().ID, 10))
	return c.Send("✅ Платёж успешно завершен!", btns)
}

func (p *Payments) PaginationHandler(action string) func(c telebot.Context) error {
	return func(c telebot.Context) error {
		st, exists := p.paginationState.Get(strconv.FormatInt(c.Sender().ID, 10))
		if !exists {
			st = state.PaginationState{
				CurrentPage: 1,
				TotalPages:  1,
			}
		}

		newPage := st.CurrentPage
		switch action {
		case "first":
			newPage = 1
		case "prev":
			newPage = max(1, st.CurrentPage-1)
		case "next":
			newPage = min(st.TotalPages, st.CurrentPage+1)
		case "last":
			newPage = st.TotalPages
		}

		return p.HistoryPaymentsHandler(c, newPage, false)
	}
}

func (p *Payments) HistoryPaymentsHandler(c telebot.Context, currentPage int, isFirst bool) error {
	defer func(c telebot.Context) {
		err := c.Respond()
		if err != nil {
			p.log.Error("Failed to send message", err)
		}
	}(c)

	const pageSize = 10
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

	p.paginationState.Set(strconv.FormatInt(c.Sender().ID, 10), state.PaginationState{
		CurrentPage: currentPage,
		TotalPages:  int(totalPages),
	})

	if currentPage < 1 || currentPage > int(totalPages) {
		return nil
	}

	offset := (currentPage - 1) * pageSize
	payments, err := p.ps.GetAll(c.Sender().ID, offset, pageSize)
	if err != nil {
		return c.Send(constants.UserError, btns)
	}

	msg := fmt.Sprintf("🧾 История платежей (страница %d из %d):\n", currentPage, totalPages)
	for i, payment := range payments {
		amount := fmt.Sprintf("+%.f", payment.Amount)
		if payment.Type == "expense" {
			amount = fmt.Sprintf("-%.f", payment.Amount)
		}

		msg += fmt.Sprintf("%d) %s RUB - %s - %s\n", i+1+offset, amount, payment.Date.Format("2006-01-02 15:04:05"), payment.Note)
	}

	btns = &telebot.ReplyMarkup{}
	if totalPages > 1 {
		btns = p.historyPgBtns.AddBtns()
	}

	if isFirst {
		return c.Send(msg, btns)
	}
	return c.EditOrSend(msg, btns)
}
