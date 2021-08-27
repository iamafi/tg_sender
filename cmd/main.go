package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"net/http"
	"os"
	"os/signal"
	"tg_sender/config"
	"tg_sender/pkg/logger"
	"tg_sender/pkg/rabbit"
	"tg_sender/rest"
	"tg_sender/service"
	"tg_sender/workers"
	"time"
)

func main() {
	quitSignal := make(chan os.Signal, 1)
	signal.Notify(quitSignal)

	// Load config and logger
	cfg := config.Load()
	log := logger.New(cfg.LogLevel, "tg_sender")
	log.Info("Config loaded - Logger Initialized")

	// Initialize RabbitMQ connection
	conn, err := rabbit.Dial(cfg.RabbitURL, log, time.Second*10)
	if err != nil {
		log.Fatal(fmt.Sprintf("RabbitMQ dialing error: %v", err))
	}

	defer func() {
		log.Info("Closing connection")

		if err := conn.Close(); err != nil {
			log.Error(err.Error())
		}
	}()

	// Consumer initialization
	arrivalConsumer, err := rabbit.NewConsumer(conn, cfg.Exchange, cfg.PublishingRoutingKey, cfg.WorkerQueueName, log)
	if err != nil {
		log.Error("AMQP consumer creating error: " + err.Error())
		return
	}

	// Telegram bot initialization
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Error("Bot cannot be started")
	}
	bot.Debug = true

	// Worker
	updater := workers.New(arrivalConsumer, log, cfg, service.New(cfg, log, *bot))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Info("Starting RabbitMQ worker")

		if err := updater.Run(ctx); err != nil {
			log.Error("On worker starting error: " + err.Error())
		}
	}()

	// REST Init
	r := gin.Default()
	handler := rest.New(cfg, log, r, conn)
	srv := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: handler,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error(fmt.Sprintf("main: Failed To Start REST Server: %s\n", err.Error()))
		}
	}()
	log.Info("main: REST Server started at port " + cfg.HTTPPort)

	defer func() {
		log.Info("Closing consumer")

		if err := arrivalConsumer.Close(); err != nil {
			log.Error("On closing consumer error: " + err.Error())
		}
	}()

	// Graceful shutdown on ctrl+c
	OSCall := <-quitSignal
	log.Debug(OSCall.String())

	cancel()
	log.Info("main: " + OSCall.String())
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("main: REST Server Graceful Shutdown Failed: " + err.Error())
	}
}