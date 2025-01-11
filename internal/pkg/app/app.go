package app

import (
	"fmt"
	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
	"log"
	"nsvpn/config"
	"nsvpn/internal/app/handlers"
	"nsvpn/internal/app/middleware"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/db"
	"nsvpn/pkg/logger"
	"time"
)

type App struct {
	cfg *config.Configuration

	paymentsRepository      *repository.Payments
	promocodesRepository    *repository.Promocodes
	serversRepository       *repository.Servers
	serverStatsRepository   *repository.ServerStats
	subscriptionsRepository *repository.Subscriptions
	usersRepository         *repository.Users

	AcceptOfferButtons, ClientButtons, ClientButtonsWithSub, ListSubscriptions *services.Buttons

	baseService          *services.Base
	paymentsService      *services.Payments
	subscriptionsService *services.Subscriptions
	serversService       *services.Servers
	usersService         *services.Users

	usersMiddleware *middleware.Users

	baseHandler          *handlers.Base
	paymentsHandler      *handlers.Payments
	promocodesHandler    *handlers.Promocodes
	subscriptionsHandler *handlers.Subscriptions
	serversHandler       *handlers.Servers
	usersHandler         *handlers.Users
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
	a := &App{}

	// конфиг
	a.cfg = cfg

	// репозитории
	a.paymentsRepository = repository.NewPayments()
	a.promocodesRepository = repository.NewPromocodes()
	a.serversRepository = repository.NewServers()
	a.serverStatsRepository = repository.NewServerStats()
	a.subscriptionsRepository = repository.NewSubscriptions()
	a.usersRepository = repository.NewUsers()

	// кнопки
	a.AcceptOfferButtons = services.NewButtons(models.AcceptOfferButton, []int{1}, "inline")
	a.ClientButtons = services.NewButtons(models.ClientButtons, []int{1, 2}, "reply")
	a.ClientButtonsWithSub = services.NewButtons(models.ClientButtonsWithSub, []int{1, 2}, "reply")
	a.ListSubscriptions = services.NewButtons(models.ListSubscriptions, []int{1, 1, 1}, "inline")

	// сервисы
	a.baseService = services.NewBase()
	a.paymentsService = services.NewPayments(a.paymentsRepository)
	a.subscriptionsService = services.NewSubscriptions(a.subscriptionsRepository)
	a.usersService = services.NewUsers(a.usersRepository)
	a.serversService = services.NewServers(a.serversRepository)

	// middleware
	a.usersMiddleware = middleware.NewUsers(a.usersRepository, a.subscriptionsService)

	// эндпоинты
	a.baseHandler = handlers.NewBase(a.AcceptOfferButtons, a.ClientButtons, a.ClientButtonsWithSub, a.usersService, a.subscriptionsService)
	a.paymentsHandler = handlers.NewPayments(a.paymentsService, a.subscriptionsService)
	//a.promocodesHandler = handlers.NewPromocodes()
	a.subscriptionsHandler = handlers.NewSubscriptions(a.ListSubscriptions)
	a.serversHandler = handlers.NewServers(a.serversService)
	//a.usersHandler = handlers.NewUsers()

	return a
}

func RunBot(a *App) error {
	pref := telebot.Settings{
		Token:  a.cfg.TelegramAPI,
		Poller: &telebot.LongPoller{Timeout: 1 * time.Second},
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		return err
	}

	b.Use(a.usersMiddleware.IsUser)
	acceptOfferBtns := a.AcceptOfferButtons.GetBtns()
	listSubsBtns := a.ListSubscriptions.GetBtns()

	b.Handle("/start", a.baseHandler.StartHandler)
	b.Handle("/help", a.baseHandler.HelpHandler)
	b.Handle(acceptOfferBtns["accept_offer"], a.baseHandler.AcceptOfferHandler)

	b.Handle("Подключить VPN", a.subscriptionsHandler.ChooseDurationHandler)
	b.Handle(listSubsBtns["sub_one_month"], a.paymentsHandler.PaymentHandler)
	b.Handle(listSubsBtns["sub_three_month"], a.paymentsHandler.PaymentHandler)
	b.Handle(listSubsBtns["sub_six_month"], a.paymentsHandler.PaymentHandler)

	b.Handle("/pay", a.paymentsHandler.PaymentHandler)
	b.Handle(telebot.OnCheckout, a.paymentsHandler.PreCheckoutHandler)

	b.Handle("Список серверов", a.serversHandler.ListCountries)

	b.Handle("На главную", a.baseHandler.StartHandler)
	b.Handle(telebot.OnText, a.baseHandler.OnTextHandler)

	logger.Debug("бот запущен")
	b.Start()
	return nil
}
