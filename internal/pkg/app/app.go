package app

import (
	"fmt"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
	"log"
	"nsvpn/config"
	"nsvpn/internal/app/handlers/base"
	"nsvpn/internal/app/handlers/payments"
	"nsvpn/internal/app/handlers/servers"
	"nsvpn/internal/app/middleware"
	baseService "nsvpn/internal/app/services/base"
	paymentsService "nsvpn/internal/app/services/payments"
	serversService "nsvpn/internal/app/services/servers"
	subscriptionsService "nsvpn/internal/app/services/subscriptions"
	usersService "nsvpn/internal/app/services/users"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/db"
	"nsvpn/pkg/logger"
	"time"
)

type App struct {
	base          *baseService.Service
	users         *usersService.Service
	payments      *paymentsService.Service
	subscriptions *subscriptionsService.Service
	servers       *serversService.Service
}

func New() (*App, error) {
	a := &App{}

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal("Ошибка при попытке спарсить .env файл в структуру", err.Error())
	}

	logger.Init(cfg.LoggerLevel)

	err = cache.Init(fmt.Sprintf("%s:%s", cfg.Redis.RedisAddr, cfg.Redis.RedisPort), cfg.Redis.RedisUsername, cfg.Redis.RedisPassword, cfg.Redis.RedisDBId)
	if err != nil {
		logger.Error("Ошибка при инициализации кэша", zap.Error(err))
	}

	err = db.Init(cfg.DB.DBUser, cfg.DB.DBPassword, cfg.DB.DBHost, cfg.DB.DBName)
	if err != nil {
		logger.Fatal("Ошибка при инициализации БД", zap.Error(err))
	}

	InitBot(cfg.TelegramAPI, a)

	return a, nil
}

func InitBot(TelegramAPI string, a *App) {
	pref := tele.Settings{
		Token:  TelegramAPI,
		Poller: &tele.LongPoller{Timeout: 1 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		logger.Fatal("Ошибка при создании бота", zap.Error(err), zap.Any("pref", pref))
	}

	// Сервисы
	a.base = baseService.New()
	a.users = usersService.New()
	a.payments = paymentsService.New()
	a.subscriptions = subscriptionsService.New()
	a.servers = serversService.New()

	// Middleware
	mw := middleware.Endpoint{Bot: b, User: a.users}
	b.Use(mw.IsUser)

	// Эндпоинты
	baseEndpoint := base.Endpoint{Base: a.base}
	//usersEndpoint := users.Endpoint{User: a.users}
	paymentsEndpoint := payments.Endpoint{Bot: b, Payments: a.payments, Subscriptions: a.subscriptions}
	serversEndpoint := servers.Endpoint{Server: a.servers}

	// Обработчики
	b.Handle("/help", baseEndpoint.HelpHandler)
	b.Handle("/pay", paymentsEndpoint.PaymentHandler)              // Получение инвойса на оплату
	b.Handle(tele.OnCheckout, paymentsEndpoint.PreCheckoutHandler) // Подтверждение оплаты
	b.Handle("/addserv", serversEndpoint.AddServerHandler)
	b.Handle("/addcl", serversEndpoint.AddClientHandler)

	logger.Debug("Бот запущен")
	b.Start()
}
