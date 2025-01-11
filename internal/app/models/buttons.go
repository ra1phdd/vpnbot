package models

type ButtonOption struct {
	Value   string
	Display string
}

var AcceptOfferButton = []ButtonOption{
	{
		Value:   "accept_offer",
		Display: "Принять условия",
	},
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
