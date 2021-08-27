package controllers

import (
	"tg_sender/pkg/logger"
	"tg_sender/service"
)

type ControllerSend struct {
	log logger.Logger
	bot service.BotService
}

func Init(log logger.Logger, bot service.BotService) ControllerSend {
	return ControllerSend{
		log: log,
		bot: bot,
	}
}

func (c ControllerSend) SendMessages(m string) error {
	err := c.bot.SendMessage(m)
	if err != nil {
		return err
	}
	return nil
}
