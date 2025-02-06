package app

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gopkg.in/telebot.v4"
	"nsvpn/internal/app/config"
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
	cfg    *config.Configuration
	bot    *telebot.Bot
	router *gin.Engine

	countryRepository       *repository.Country
	currencyRepository      *repository.Currency
	paymentsRepository      *repository.Payments
	promocodesRepository    *repository.Promocodes
	serversRepository       *repository.Servers
	serverStatsRepository   *repository.ServerStats
	subscriptionsRepository *repository.Subscriptions
	keysRepository          *repository.Keys
	usersRepository         *repository.Users

	AcceptOfferButtons, ClientButtons, ClientButtonsWithSub, ListSubscriptions *services.Buttons

	baseService          *services.Base
	countryService       *services.Country
	paymentsService      *services.Payments
	subscriptionsService *services.Subscriptions
	serversService       *services.Servers
	keysService          *services.Keys
	usersService         *services.Users

	usersMiddleware *middleware.Users

	baseHandler          *handlers.Base
	paymentsHandler      *handlers.Payments
	promocodesHandler    *handlers.Promocodes
	subscriptionsHandler *handlers.Subscriptions
	serversHandler       *handlers.Servers
	keysHandler          *handlers.Keys
	usersHandler         *handlers.Users
}

func New() error {
	logger.Init()

	cfg, err := config.NewConfig()
	if err != nil {
		logger.Error("Error loading config from env", zap.Error(err))
		return err
	}
	logger.SetLogLevel(cfg.LoggerLevel)

	err = db.Init(cfg.DB.DBUser, cfg.DB.DBPassword, cfg.DB.DBHost, cfg.DB.DBName)
	if err != nil {
		logger.Error("DB initialization failed", zap.Error(err))
		return err
	}

	err = cache.Init(fmt.Sprintf("%s:%s", cfg.Redis.RedisAddr, cfg.Redis.RedisPort), cfg.Redis.RedisUsername, cfg.Redis.RedisPassword, cfg.Redis.RedisDBId)
	if err != nil {
		logger.Error("Redis cache initialization failed", zap.Error(err))
	}

	a := setupApplication(cfg)

	go setupServer(a)
	return setupBot(a)
}

func setupApplication(cfg *config.Configuration) *App {
	a := &App{}

	// cfg
	a.cfg = cfg

	// repo
	a.countryRepository = repository.NewCountry()
	a.currencyRepository = repository.NewCurrency()
	a.paymentsRepository = repository.NewPayments()
	a.promocodesRepository = repository.NewPromocodes()
	a.serversRepository = repository.NewServers()
	a.serverStatsRepository = repository.NewServerStats()
	a.subscriptionsRepository = repository.NewSubscriptions()
	a.keysRepository = repository.NewKeys()
	a.usersRepository = repository.NewUsers()

	// buttons
	a.AcceptOfferButtons = services.NewButtons(models.AcceptOfferButton, []int{1}, "inline")
	a.ClientButtons = services.NewButtons(models.ClientButtons, []int{1, 2}, "reply")
	a.ClientButtonsWithSub = services.NewButtons(models.ClientButtonsWithSub, []int{1, 2}, "reply")
	a.ListSubscriptions = services.NewButtons(models.ListSubscriptions, []int{1, 1, 1}, "inline")

	// services
	a.baseService = services.NewBase()
	a.countryService = services.NewCountry(a.countryRepository)
	a.paymentsService = services.NewPayments(a.paymentsRepository, a.currencyRepository)
	a.subscriptionsService = services.NewSubscriptions(a.subscriptionsRepository)
	a.usersService = services.NewUsers(a.usersRepository)
	a.keysService = services.NewKeys(a.keysRepository)
	a.serversService = services.NewServers(a.serversRepository, a.countryRepository)

	// middleware
	a.usersMiddleware = middleware.NewUsers(a.usersRepository, a.subscriptionsService)

	// handlers
	a.keysHandler = handlers.NewKeys(a.keysService, a.serversService, a.subscriptionsService)
	a.subscriptionsHandler = handlers.NewSubscriptions(a.ListSubscriptions)
	a.serversHandler = handlers.NewServers(a.bot, a.keysHandler, a.countryService, a.serversService)
	a.baseHandler = handlers.NewBase(a.AcceptOfferButtons, a.ClientButtons, a.ClientButtonsWithSub, a.usersService, a.subscriptionsService, a.serversHandler)
	a.paymentsHandler = handlers.NewPayments(a.paymentsService, a.subscriptionsService)
	a.promocodesHandler = handlers.NewPromocodes(a.usersService)
	a.usersHandler = handlers.NewUsers()

	return a
}

func setupBot(a *App) (err error) {
	a.bot, err = telebot.NewBot(telebot.Settings{
		Token:  a.cfg.TelegramAPI,
		Poller: &telebot.LongPoller{Timeout: 1 * time.Second},
	})
	if err != nil {
		logger.Error("Error setting up telegram bot", zap.Error(err))
		return err
	}

	a.bot.Use(a.usersMiddleware.IsUser)
	acceptOfferBtns := a.AcceptOfferButtons.GetBtns()
	listSubsBtns := a.ListSubscriptions.GetBtns()

	a.bot.Handle("/start", a.baseHandler.StartHandler)
	a.bot.Handle("/help", a.baseHandler.HelpHandler)
	a.bot.Handle(acceptOfferBtns["accept_offer"], a.baseHandler.AcceptOfferHandler)

	a.bot.Handle("Подключить VPN", a.subscriptionsHandler.ChooseDurationHandler)
	a.bot.Handle(listSubsBtns["sub_one_month"], a.paymentsHandler.PaymentHandler)
	a.bot.Handle(listSubsBtns["sub_three_month"], a.paymentsHandler.PaymentHandler)
	a.bot.Handle(listSubsBtns["sub_six_month"], a.paymentsHandler.PaymentHandler)

	a.bot.Handle("/pay", a.paymentsHandler.PaymentHandler)
	a.bot.Handle(telebot.OnCheckout, a.paymentsHandler.PreCheckoutHandler)

	a.bot.Handle("Список серверов", a.serversHandler.ListCountriesHandler)

	a.bot.Handle("Назад", a.baseHandler.StartHandler)
	a.bot.Handle(telebot.OnText, a.baseHandler.OnTextHandler)

	a.bot.Start()
	return nil
}

func setupServer(a *App) {
	gin.SetMode(a.cfg.GinMode)
	a.router = gin.Default()

	// регистрируем маршруты
	//r.GET("/v1/client/is_found", endpointClient.IsFound)

	err := a.router.Run(fmt.Sprintf(":%d", a.cfg.Port))

	if err != nil {
		logger.Fatal("server startup error", zap.Error(err))
	}
}
