package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cast"
)

type Config struct {
	Environment          string
	LogLevel             string
	RabbitURL            string
	PublishingRoutingKey string
	WorkerRoutingKey     string
	WorkerQueueName      string
	BotToken             string
	ChatID               int64
	Exchange             string
	HTTPPort          string
}

func Load() Config {
	_ = godotenv.Load()

	config := Config{
		Environment:          cast.ToString(env("ENVIRONMENT", "development")),
		LogLevel:             cast.ToString(env("LOG_LEVEL", "debug")),
		RabbitURL:            cast.ToString(env("AMQP_URL", "amqp://guest:guest@localhost:5672/")),
		PublishingRoutingKey: cast.ToString(env("PUBLISHING_ROUTING_KEY", "telegram.sent")),
		WorkerRoutingKey:     cast.ToString(env("WORKER_ROUTING_KEY", "sent.telegram")),
		WorkerQueueName:      cast.ToString(env("WORKER_QUEUE_NAME", "tg_sender")),
		BotToken: 			  cast.ToString(env("BOT_TOKEN", "bot_token")), //set your token
		ChatID: 			  cast.ToInt64(env("CHAT_ID", 0)), //set your chat id
		Exchange:             cast.ToString(env("EXCHANGE", "sent")),
		HTTPPort:          cast.ToString(env("HTTP_PORT", ":80")),
	}

	return config
}

func env(key string, defaultValue interface{}) interface{} {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}

	return defaultValue
}