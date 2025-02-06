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
		Display: "📡 Подключить VPN",
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

var ClientButtonsWithSub = []ButtonOption{
	{
		Value:   "listservers",
		Display: "Список серверов",
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

var ListSubscriptions = []ButtonOption{
	{
		Value:   "sub_one_month",
		Display: "1 месяц (149₽)",
	},
	{
		Value:   "sub_three_month",
		Display: "3 месяца (399₽ | -11%)",
	},
	{
		Value:   "sub_six_month",
		Display: "6 месяцев (749₽ | -17%)",
	},
}

var ListServers = []ButtonOption{
	{
		Value:   "listservers",
		Display: "Список серверов",
	},
}
