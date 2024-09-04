package constants

import "fmt"

var (
	ErrUserNotFound    = fmt.Errorf("пользователь не найден")
	ErrSubNotFound     = fmt.Errorf("подписка не найдена")
	ErrPaymentNotFound = fmt.Errorf("платеж не найден")
	ErrServerNotFound  = fmt.Errorf("сервер не найден")
)
