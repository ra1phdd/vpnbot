package constants

import (
	"errors"
)

var (
	ErrInsufficientFunds  = errors.New("insufficient funds on the user's balance")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmptyFields        = errors.New("one of the fields is empty")
	ErrPaymentTimeExpired = errors.New("payment time expired")
)

const (
	UserHasNoRights  = "❌ У вас нет прав на выполнение данной команды"
	UserError        = "❌ Упс! Что-то сломалось. Повторите попытку чуть позже или обратитесь в службу поддержки"
	UserLittleAmount = "❌ Сумма пополнения не может быть меньше 10 RUB"
)
