package app

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"gopkg.in/telebot.v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"nsvpn/internal/app/config"
	"nsvpn/internal/app/handlers"
	"nsvpn/internal/app/middleware"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/logger"
	"time"
)

type App struct {
	cfg   *config.Configuration
	log   *logger.Logger
	db    *gorm.DB
	redis *redis.Client
	bot   *telebot.Bot

	countryRepo       *repository.Country
	currencyRepo      *repository.Currency
	paymentsRepo      *repository.Payments
	promocodesRepo    *repository.Promocodes
	serversRepo       *repository.Servers
	subscriptionsRepo *repository.Subscriptions
	keysRepo          *repository.Keys
	usersRepo         *repository.Users

	AcceptOfferButtons   *services.Buttons
	ClientButtons        *services.Buttons
	ClientButtonsWithSub *services.Buttons
	ListSubscriptions    *services.Buttons

	baseService          *services.Base
	countryService       *services.Country
	currencyService      *services.Currency
	paymentsService      *services.Payments
	promocodesService    *services.Promocodes
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
	a := &App{
		log: logger.New(),
	}

	var err error
	a.cfg, err = config.NewConfig()
	if err != nil {
		a.log.Error("Error loading config from env", err)
		return err
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Europe/Moscow",
		a.cfg.DB.Address, a.cfg.DB.Username, a.cfg.DB.Password, a.cfg.DB.Name, a.cfg.DB.Port)
	a.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	err = a.db.AutoMigrate(
		&models.User{},
		&models.Partner{},
		&models.Country{},
		&models.Server{},
		&models.Subscription{},
		&models.Currency{},
		&models.Payment{},
		&models.Key{},
		&models.Promocode{},
		&models.PromocodeActivations{},
	)
	if err != nil {
		return err
	}

	a.bot, err = telebot.NewBot(telebot.Settings{
		Token:  a.cfg.TelegramAPI,
		Poller: &telebot.LongPoller{Timeout: 1 * time.Second},
	})
	if err != nil {
		a.log.Error("Failed creating telegram bot", err)
		return err
	}

	a.redis = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", a.cfg.Redis.Address, a.cfg.Redis.Port),
		Username: a.cfg.Redis.Username,
		Password: a.cfg.Redis.Password,
		DB:       a.cfg.Redis.DB,
	})
	err = a.redis.Ping(context.Background()).Err()
	if err != nil {
		return err
	}

	a.redis.FlushAll(context.Background())

	// repo
	a.countryRepo = repository.NewCountry(a.log, a.db, a.redis)
	a.currencyRepo = repository.NewCurrency(a.log, a.db, a.redis)
	a.paymentsRepo = repository.NewPayments(a.log, a.db, a.redis)
	a.promocodesRepo = repository.NewPromocodes(a.log, a.db, a.redis)
	a.serversRepo = repository.NewServers(a.log, a.db, a.redis)
	a.subscriptionsRepo = repository.NewSubscriptions(a.log, a.db, a.redis)
	a.keysRepo = repository.NewKeys(a.log, a.db, a.redis)
	a.usersRepo = repository.NewUsers(a.log, a.db, a.redis)

	// buttons
	a.AcceptOfferButtons = services.NewButtons(models.AcceptOfferButton, []int{1}, "inline")
	a.ClientButtons = services.NewButtons(models.ClientButtons, []int{1, 2}, "reply")
	a.ClientButtonsWithSub = services.NewButtons(models.ClientButtonsWithSub, []int{1, 2}, "reply")
	a.ListSubscriptions = services.NewButtons(models.ListSubscriptions, []int{1, 1, 1}, "inline")

	// services
	a.baseService = services.NewBase(a.log)
	a.countryService = services.NewCountry(a.log, a.countryRepo)
	a.currencyService = services.NewCurrency(a.log, a.currencyRepo)
	a.paymentsService = services.NewPayments(a.log, a.paymentsRepo)
	a.subscriptionsService = services.NewSubscriptions(a.log, a.subscriptionsRepo)
	a.usersService = services.NewUsers(a.log, a.usersRepo)
	a.keysService = services.NewKeys(a.log, a.keysRepo)
	a.serversService = services.NewServers(a.log, a.serversRepo)

	// middleware
	a.usersMiddleware = middleware.NewUsers(a.log, a.usersRepo, a.subscriptionsService)

	// handlers
	a.keysHandler = handlers.NewKeys(a.log, a.keysService, a.subscriptionsService)
	a.subscriptionsHandler = handlers.NewSubscriptions(a.log, a.ListSubscriptions)
	a.serversHandler = handlers.NewServers(a.log, a.bot, a.serversService, a.keysHandler, a.countryService)
	a.baseHandler = handlers.NewBase(a.log, a.AcceptOfferButtons, a.ClientButtons, a.ClientButtonsWithSub, a.usersService, a.subscriptionsService, a.serversHandler)
	a.paymentsHandler = handlers.NewPayments(a.log, a.paymentsService, a.currencyService, a.subscriptionsService, a.ClientButtonsWithSub)
	a.promocodesHandler = handlers.NewPromocodes(a.log, a.promocodesService)
	a.usersHandler = handlers.NewUsers(a.log)

	return RunBot(a)
}

func RunBot(a *App) error {
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
