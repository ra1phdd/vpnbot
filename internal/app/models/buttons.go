package models

type ButtonOption struct {
	Value   string
	Display string
}

var AcceptOfferButton = []ButtonOption{
	{
		Value:   "accept_offer",
		Display: "✅ Принять условия",
	},
}

var ClientButtons = []ButtonOption{
	{
		Value:   "attach_vpn",
		Display: "🔒 Подключить VPN",
	},
	{
		Value:   "profile",
		Display: "👔 Профиль",
	},
	{
		Value:   "technical_support",
		Display: "💬 Техподдержка",
	},
	{
		Value:   "info",
		Display: "💡 Информация",
	},
}

var ClientButtonsWithSub = []ButtonOption{
	{
		Value:   "list_servers",
		Display: "🌐 Список серверов",
	},
	{
		Value:   "profile",
		Display: "👔 Профиль",
	},
	{
		Value:   "technical_support",
		Display: "💬 Техподдержка",
	},
	{
		Value:   "info",
		Display: "💡 Информация",
	},
}
