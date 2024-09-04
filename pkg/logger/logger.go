package logger

import (
	"fmt"
	"gopkg.in/telebot.v3"
	"log"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func Init(loggerLevel string) {
	customTimeEncoder := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05"))
	}

	config := zap.NewProductionEncoderConfig()
	config.EncodeLevel = zapcore.LowercaseLevelEncoder
	config.EncodeTime = customTimeEncoder

	fileEncoder := zapcore.NewJSONEncoder(config)
	consoleEncoder := zapcore.NewConsoleEncoder(config)

	err := os.MkdirAll("logs", os.ModePerm)
	if err != nil {
		log.Fatal("Ошибка создания папки logs", err.Error())
	}
	logFile, err := os.OpenFile("logs/main.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Ошибка создания файла main.log", err.Error())
	}
	writer := zapcore.AddSync(logFile)

	var defaultLogLevel zapcore.Level
	switch loggerLevel {
	case "debug":
		defaultLogLevel = zapcore.DebugLevel
	case "warn":
		defaultLogLevel = zapcore.WarnLevel
	case "error":
		defaultLogLevel = zapcore.ErrorLevel
	case "fatal":
		defaultLogLevel = zapcore.FatalLevel
	case "info":
	default:
		defaultLogLevel = zapcore.InfoLevel
	}

	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, writer, defaultLogLevel),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), defaultLogLevel),
	)
	logger = zap.New(core, zap.AddStacktrace(zapcore.ErrorLevel))
	defer logger.Sync()
}

func Debug(message string, fields ...zap.Field) {
	logger.Debug(message, fields...)
}

func Debugf(message string, c *telebot.Chat, fields ...zap.Field) {
	name := c.FirstName
	if c.Title != "" {
		name = c.Title
	} else if c.LastName != "" {
		name = fmt.Sprintf("%s %s", c.FirstName, c.LastName)
	}
	logger.Debug(fmt.Sprintf("[%d/%s] %s", c.ID, name, message), fields...)
}

func Info(message string, fields ...zap.Field) {
	logger.Info(message, fields...)
}

func Infof(message string, c *telebot.Chat, fields ...zap.Field) {
	name := c.FirstName
	if c.Title != "" {
		name = c.Title
	} else if c.LastName != "" {
		name = fmt.Sprintf("%s %s", c.FirstName, c.LastName)
	}
	logger.Info(fmt.Sprintf("[%d/%s] %s", c.ID, name, message), fields...)
}

func Warn(message string, fields ...zap.Field) {
	logger.Warn(message, fields...)
}

func Warnf(message string, c *telebot.Chat, fields ...zap.Field) {
	name := c.FirstName
	if c.Title != "" {
		name = c.Title
	} else if c.LastName != "" {
		name = fmt.Sprintf("%s %s", c.FirstName, c.LastName)
	}
	logger.Warn(fmt.Sprintf("[%d/%s] %s", c.ID, name, message), fields...)
}

func Error(message string, fields ...zap.Field) {
	logger.Error(message, fields...)
}

func Errorf(message string, c *telebot.Chat, fields ...zap.Field) {
	name := c.FirstName
	if c.Title != "" {
		name = c.Title
	} else if c.LastName != "" {
		name = fmt.Sprintf("%s %s", c.FirstName, c.LastName)
	}
	logger.Error(fmt.Sprintf("[%d/%s] %s", c.ID, name, message), fields...)
}

func Fatal(message string, fields ...zap.Field) {
	logger.Fatal(message, fields...)
}

func Fatalf(message string, c *telebot.Chat, fields ...zap.Field) {
	name := c.FirstName
	if c.Title != "" {
		name = c.Title
	} else if c.LastName != "" {
		name = fmt.Sprintf("%s %s", c.FirstName, c.LastName)
	}
	logger.Fatal(fmt.Sprintf("[%d/%s] %s", c.ID, name, message), fields...)
}
