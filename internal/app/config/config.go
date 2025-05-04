package config

import (
	"log"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
)

type Configuration struct {
	TelegramAPI string `env:"TELEGRAM_API,required"`

	YoukassaURL string `env:"YOUKASSA_URL,required"`
	YoukassaAPI string `env:"YOUKASSA_API,required"`
	YoukassaID  string `env:"YOUKASSA_ID,required"`

	HeleketURL string `env:"HELEKET_URL,required"`
	HeleketAPI string `env:"HELEKET_API,required"`
	HeleketID  string `env:"HELEKET_ID,required"`

	LoggerLevel string `env:"LOGGER_LEVEL" envDefault:"info"`
	GinMode     string `env:"GIN_MODE" envDefault:"release"`
	PortAPI     int    `env:"PORT_API" envDefault:"8890"`
	DB          DB
	Redis       Redis
}

type DB struct {
	Address  string `env:"DB_ADDR,required"`
	Port     int    `env:"DB_PORT" envDefault:"5432"`
	Username string `env:"DB_USER,required"`
	Password string `env:"DB_PASS,required"`
	Name     string `env:"DB_NAME,required"`
}

type Redis struct {
	Address  string `env:"REDIS_ADDR,required"`
	Port     int    `env:"REDIS_PORT" envDefault:"6379"`
	Username string `env:"REDIS_USER,required"`
	Password string `env:"REDIS_PASS,required"`
	DB       int    `env:"REDIS_DB,required"`
}

func NewConfig(files ...string) (*Configuration, error) {
	err := godotenv.Load(files...)
	if err != nil {
		log.Fatal("Файл .env не найден", err.Error())
	}

	cfg := Configuration{}
	err = env.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	err = env.Parse(&cfg.Redis)
	if err != nil {
		return nil, err
	}
	err = env.Parse(&cfg.DB)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
