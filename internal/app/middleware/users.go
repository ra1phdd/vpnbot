package middleware

import (
	"gopkg.in/telebot.v4"
	"log/slog"
	"nsvpn/internal/app/constants"
	"nsvpn/internal/app/handlers"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"strconv"
)

type Users struct {
	log                  *logger.Logger
	bot                  *telebot.Bot
	us                   *services.Users
	ss                   *services.Subscriptions
	clientButtons        *services.Buttons
	clientButtonsWithSub *services.Buttons
	bh                   *handlers.Base
}

func NewUsers(log *logger.Logger, bot *telebot.Bot, us *services.Users, ss *services.Subscriptions, clientButtons, clientButtonsWithSub *services.Buttons, bh *handlers.Base) *Users {
	return &Users{
		log:                  log,
		bot:                  bot,
		us:                   us,
		ss:                   ss,
		clientButtons:        clientButtons,
		clientButtonsWithSub: clientButtonsWithSub,
		bh:                   bh,
	}
}

func (u *Users) IsUser(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		user, err := u.getOrCreateUser(c)
		if err != nil {
			return err
		}

		if user.Username != c.Sender().Username || user.Firstname != c.Sender().FirstName || user.Lastname != c.Sender().LastName {
			if err := u.us.Update(c.Sender().ID, user); err != nil {
				u.log.Error("Error while updating user", err)
			}
		}
		c.Set("user", user)

		if shouldCheckSign(user, c) {
			u.isSign(c)
			return nil
		}

		sub, err := u.handleSubscription(c)
		if err != nil {
			return err
		}

		u.setSubscriptionContext(c, sub)
		return next(c)
	}
}

func (u *Users) getOrCreateUser(c telebot.Context) (*models.User, error) {
	user, err := u.us.Get(c.Sender().ID)
	if err != nil {
		_ = c.Send(constants.UserError)
		u.log.Error("Error while fetching user", err)
		return nil, err
	}

	if user == nil {
		return u.createNewUser(c)
	}
	return user, nil
}

func (u *Users) createNewUser(c telebot.Context) (*models.User, error) {
	partnerID := u.parsePartnerID(c.Data())
	if c.Sender().ID == partnerID {
		partnerID = 0
	}

	user := &models.User{
		ID:        c.Sender().ID,
		Username:  c.Sender().Username,
		Firstname: c.Sender().FirstName,
		Lastname:  c.Sender().LastName,
		PartnerID: partnerID,
		IsAdmin:   false,
		IsSign:    false,
	}

	if err := u.us.Add(user); err != nil {
		_ = c.Send(constants.UserError)
		u.log.Error("Failed to create new user", err, slog.Int64("userId", c.Sender().ID))
		return nil, err
	}
	return user, nil
}

func (u *Users) parsePartnerID(data string) int64 {
	if data == "" {
		return 0
	}
	parsedID, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return 0
	}
	return parsedID
}

func shouldCheckSign(user *models.User, c telebot.Context) bool {
	if user.IsSign {
		return false
	}
	return c.Callback() == nil ||
		(c.Callback() != nil && c.Callback().Unique != models.AcceptOfferButton[0].Value)
}

func (u *Users) handleSubscription(c telebot.Context) (*models.Subscription, error) {
	sub, err := u.ss.GetLastByUserID(c.Sender().ID, true)
	if err != nil {
		_ = c.Send(constants.UserError)
		u.log.Error("Error while fetching subscription", err)
		return nil, err
	}
	return sub, nil
}

func (u *Users) setSubscriptionContext(c telebot.Context, sub *models.Subscription) {
	u.isSubActive(c, sub)
	if sub != nil {
		c.Set("sub", sub)
	}
}

func (u *Users) isSign(c telebot.Context) {
	acceptOfferButtons := services.NewButtons(models.AcceptOfferButton, []int{1}, "inline")
	u.bot.Handle(acceptOfferButtons.GetBtn("accept_offer"), u.bh.AcceptOfferHandler)

	err := c.Send("Чтобы начать пользоваться NSVPN, необходимо принять условия публичной [оферты](https://teletype.in/@nsvpn/Dpvwcj7llQx).", acceptOfferButtons.AddBtns(), telebot.ModeMarkdown)
	if err != nil {
		u.log.Error("Error while sending message", err)
	}
}

func (u *Users) isSubActive(c telebot.Context, sub *models.Subscription) {
	buttons := u.clientButtonsWithSub.AddBtns()
	if sub == nil || !sub.IsActive {
		buttons = u.clientButtons.AddBtns()
	}
	c.Set("replyKeyboard", buttons)
}
