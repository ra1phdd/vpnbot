package models

type ButtonOption struct {
	Value   string
	Display string
}

var AcceptOfferButton = ButtonOption{
	Value:   "Принять",
	Display: "accept",
}

var ClientButtons = []ButtonOption{
	{
		Value:   "attachvpn",
		Display: "Подключить VPN",
	},
	{
		Value:   "profile",
		Display: "Профиль",
	},
	{
		Value:   "info",
		Display: "Информация",
	},
}
