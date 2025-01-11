package app

import (
	"fmt"
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
	"log"
	"nsvpn/config"
	"nsvpn/internal/app/handlers"
	"nsvpn/internal/app/middleware"
	repository2 "nsvpn/internal/app/repository"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/db"
	"nsvpn/pkg/logger"
	"time"
)

type App struct {
	cfg                     *config.Configuration
	paymentsRepository      *repository2.Payments
	promocodesRepository    *repository2.Promocodes
	serversRepository       *repository2.Servers
	serverStatsRepository   *repository2.ServerStats
	subscriptionsRepository *repository2.Subscriptions
	usersRepository         *repository2.Users
}

func New() error {
	logger.Init()

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal("Ошибка при попытке спарсить .env файл в структуру", err.Error())
	}

	err = cache.Init(fmt.Sprintf("%s:%s", cfg.Redis.RedisAddr, cfg.Redis.RedisPort), cfg.Redis.RedisUsername, cfg.Redis.RedisPassword, cfg.Redis.RedisDBId)
	if err != nil {
		logger.Error("Ошибка при инициализации кэша", zap.Error(err))
	}

	err = db.Init(cfg.DB.DBUser, cfg.DB.DBPassword, cfg.DB.DBHost, cfg.DB.DBName)
	if err != nil {
		logger.Fatal("Ошибка при инициализации БД", zap.Error(err))
	}
	logger.SetLogLevel(cfg.LoggerLevel)

	a := setupApplication(cfg)

	return RunBot(a)
}

func setupApplication(cfg *config.Configuration) *App {
	var a *App

	// конфиг
	a.cfg = cfg

	// репозитории
	a.paymentsRepository = repository2.NewPayments()
	a.promocodesRepository = repository2.NewPromocodes()
	a.serversRepository = repository2.NewServers()
	a.serverStatsRepository = repository2.NewServerStats()
	a.subscriptionsRepository = repository2.NewSubscriptions()
	a.usersRepository = repository2.NewUsers()

	return a
}

func RunBot(a *App) error {
	pref := tele.Settings{
		Token:  a.cfg.TelegramAPI,
		Poller: &tele.LongPoller{Timeout: 1 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return err
	}

	menu := &tele.ReplyMarkup{ResizeKeyboard: true, IsPersistent: true}

	// Middleware
	mw := middleware.Endpoint{Bot: b, User: a.users}
	b.Use(mw.IsUser)

	// Эндпоинты
	baseEndpoint := handlers.Endpoint{Base: a.base}
	//usersEndpoint := users.Endpoint{User: a.users}
	paymentsEndpoint := handlers.Endpoint{Bot: b, Payments: a.paymentsRepository, Subscriptions: a.subscriptionsRepository}
	serversEndpoint := handlers.Endpoint{Server: a.serversRepository}
	promocodesEndpoint := handlers.Endpoint{Promocodes: a.promocodesRepository}

	b.Handle("/start", func(c tb.Context) error {
		// Проверка на наличие реферальной ссылки
		if c.Message.ReplyTo != nil {
			// Получение ID пользователя, который отправил сообщение
			referrerID := c.Message.ReplyTo.From.ID
			fmt.Printf("Пользователь запустил бота по реферальной ссылке от пользователя ID: %d\n", referrerID)
			// Здесь можно выполнить дополнительные действия, например, записать реферала в базу данных
		}

		// Ответ пользователю
		return c.Send("Добро пожаловать! Используйте /help для получения справки.")
	})

	// Обработчики
	b.Handle("/help", baseEndpoint.HelpHandler)
	b.Handle("/pay", paymentsEndpoint.PaymentHandler)              // Получение инвойса на оплату
	b.Handle(tele.OnCheckout, paymentsEndpoint.PreCheckoutHandler) // Подтверждение оплаты

	// тестовые обработчики, в дальнейшем переделать
	b.Handle("/addserv", serversEndpoint.AddServerHandler)
	b.Handle("/addcl", serversEndpoint.AddClientHandler)
	b.Handle("/getserv", serversEndpoint.GetServerHandler)
	b.Handle("/getpromo", promocodesEndpoint.GetPromocodesHandler)

	logger.Debug("Бот запущен")
	b.Start()

	return nil
}
