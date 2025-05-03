package app

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"gopkg.in/telebot.v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"nsvpn/internal/app/api"
	"nsvpn/internal/app/config"
	"nsvpn/internal/app/handlers"
	"nsvpn/internal/app/middleware"
	"nsvpn/internal/app/models"
	"nsvpn/internal/app/repository"
	"nsvpn/internal/app/services"
	"nsvpn/pkg/cache"
	"nsvpn/pkg/logger"
	"time"
)

type App struct {
	cfg   *config.Configuration
	log   *logger.Logger
	db    *gorm.DB
	cache *cache.Cache
	bot   *telebot.Bot
	api   *api.API

	countryRepo       *repository.Country
	currencyRepo      *repository.Currency
	keysRepo          *repository.Keys
	paymentsRepo      *repository.Payments
	promocodesRepo    *repository.Promocodes
	serversRepo       *repository.Servers
	subscriptionsRepo *repository.Subscriptions
	usersRepo         *repository.Users

	clientButtons        *services.Buttons
	clientButtonsWithSub *services.Buttons

	baseService          *services.Base
	checkService         *services.Check
	countryService       *services.Country
	currencyService      *services.Currency
	keysService          *services.Keys
	paymentsService      *services.Payments
	promocodesService    *services.Promocodes
	serversService       *services.Servers
	subscriptionsService *services.Subscriptions
	usersService         *services.Users

	usersMiddleware *middleware.Users

	baseHandler          *handlers.Base
	keysHandler          *handlers.Keys
	paymentsHandler      *handlers.Payments
	promocodesHandler    *handlers.Promocodes
	serversHandler       *handlers.Servers
	subscriptionsHandler *handlers.Subscriptions
	usersHandler         *handlers.Users
}

func New() error {
	a := &App{
		log: logger.New(),
	}

	if err := a.initConfig(); err != nil {
		return err
	}

	if err := a.initDB(); err != nil {
		return err
	}

	if err := a.initCache(); err != nil {
		return err
	}

	if err := a.initBot(); err != nil {
		return err
	}

	a.clientButtons = services.NewButtons(models.ClientButtons, []int{1, 2}, "reply")
	a.clientButtonsWithSub = services.NewButtons(models.ClientButtonsWithSub, []int{1, 2}, "reply")

	a.api = api.NewAPI(a.log)
	a.initRepo()
	a.initServices()
	a.initHandlers()
	a.initMiddlewares()

	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		a.checkService.Run()
		for range ticker.C {
			a.checkService.Run()
		}
	}()

	return a.run()
}

func (a *App) initConfig() (err error) {
	a.cfg, err = config.NewConfig()
	if err != nil {
		a.log.Error("Error loading config from env", err)
		return err
	}
	a.log.SetLogLevel(a.cfg.LoggerLevel)
	return nil
}

func (a *App) initDB() (err error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Europe/Moscow",
		a.cfg.DB.Address, a.cfg.DB.Username, a.cfg.DB.Password, a.cfg.DB.Name, a.cfg.DB.Port)
	a.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	return a.db.AutoMigrate(
		&models.User{},
		&models.Country{},
		&models.Server{},
		&models.Subscription{},
		&models.SubscriptionPlan{},
		&models.SubscriptionPrice{},
		&models.Currency{},
		&models.Payment{},
		&models.Key{},
		&models.Promocode{},
		&models.PromocodeActivations{},
	)
}

func (a *App) initCache() error {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", a.cfg.Redis.Address, a.cfg.Redis.Port),
		Username: a.cfg.Redis.Username,
		Password: a.cfg.Redis.Password,
		DB:       a.cfg.Redis.DB,
	})
	err := client.Ping(ctx).Err()
	if err != nil {
		return err
	}
	client.FlushAll(ctx)

	a.cache = cache.New(a.log, client)
	return nil
}

func (a *App) initBot() (err error) {
	a.bot, err = telebot.NewBot(telebot.Settings{
		Token:  a.cfg.TelegramAPI,
		Poller: &telebot.LongPoller{Timeout: 1 * time.Second},
	})
	if err != nil {
		a.log.Error("Failed creating telegram bot", err)
		return err
	}

	return nil
}

func (a *App) initRepo() {
	a.countryRepo = repository.NewCountry(a.log, a.db, a.cache)
	a.currencyRepo = repository.NewCurrency(a.log, a.db, a.cache)
	a.keysRepo = repository.NewKeys(a.log, a.db, a.cache)
	a.paymentsRepo = repository.NewPayments(a.log, a.db, a.cache)
	a.promocodesRepo = repository.NewPromocodes(a.log, a.db, a.cache)
	a.serversRepo = repository.NewServers(a.log, a.db, a.cache)
	a.subscriptionsRepo = repository.NewSubscriptions(a.log, a.db, a.cache)
	a.usersRepo = repository.NewUsers(a.log, a.db, a.cache)
}

func (a *App) initServices() {
	a.baseService = services.NewBase(a.log)
	a.countryService = services.NewCountry(a.log, a.countryRepo)
	a.currencyService = services.NewCurrency(a.log, a.currencyRepo)
	a.paymentsService = services.NewPayments(a.log, a.paymentsRepo, a.currencyRepo)
	a.promocodesService = services.NewPromocodes(a.log, a.promocodesRepo)
	a.subscriptionsService = services.NewSubscriptions(a.log, a.subscriptionsRepo)
	a.usersService = services.NewUsers(a.log, a.usersRepo)
	a.keysService = services.NewKeys(a.log, a.keysRepo)
	a.serversService = services.NewServers(a.log, a.serversRepo)
	a.checkService = services.NewCheck(a.log, a.bot, a.keysService, a.subscriptionsService, a.serversService, a.usersService, a.api, a.clientButtons)
}

func (a *App) initMiddlewares() {
	a.usersMiddleware = middleware.NewUsers(a.log, a.bot, a.usersService, a.subscriptionsService, a.clientButtons, a.clientButtonsWithSub, a.baseHandler)
}

func (a *App) initHandlers() {
	a.usersHandler = handlers.NewUsers(a.log, a.usersService)
	a.keysHandler = handlers.NewKeys(a.log, a.bot, a.keysService, a.serversService, a.subscriptionsService, a.api)
	a.paymentsHandler = handlers.NewPayments(a.log, a.bot, a.paymentsService, a.currencyService, a.usersService)
	a.subscriptionsHandler = handlers.NewSubscriptions(a.log, a.bot, a.subscriptionsService, a.countryService, a.currencyService, a.paymentsService, a.usersService, a.paymentsHandler, a.clientButtonsWithSub)
	a.serversHandler = handlers.NewServers(a.log, a.bot, a.serversService, a.keysHandler, a.countryService, a.api)
	a.baseHandler = handlers.NewBase(a.log, a.bot, a.usersService, a.subscriptionsHandler, a.paymentsHandler)
	a.promocodesHandler = handlers.NewPromocodes(a.log, a.promocodesService, a.usersService)
}

func (a *App) run() error {
	a.bot.Use(a.usersMiddleware.IsUser)

	a.bot.Handle("/start", a.baseHandler.StartHandler)
	a.bot.Handle("/help", a.baseHandler.HelpHandler)
	a.bot.Handle("👔 Профиль", a.baseHandler.ProfileHandler)
	a.bot.Handle("💡 Информация", a.baseHandler.InfoHandler)
	a.bot.Handle("🔒 Подключить VPN", a.subscriptionsHandler.ChooseDurationHandler)
	a.bot.Handle("🌐 Список серверов", a.serversHandler.ListCountriesHandler)

	a.bot.Handle(telebot.OnText, a.baseHandler.OnTextHandler)

	a.bot.Start()
	return nil
}
