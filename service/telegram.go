package service

import (
	"errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"tg_sender/config"
	"tg_sender/pkg/logger"
)

type BotService struct {
	config  config.Config
	log     logger.Logger
	bot     tgbotapi.BotAPI
}

func New(config config.Config, log logger.Logger, bot tgbotapi.BotAPI) BotService {
	return BotService{
		config: config,
		log: log,
		bot: bot,
	}
}

func (s BotService) SendMessage(text string) error {
	msg := tgbotapi.NewMessage(s.config.ChatID, text)
	_, err := s.bot.Send(msg)
	if err != nil {
		return errors.New("cannot send a message")
	}
	return nil
}