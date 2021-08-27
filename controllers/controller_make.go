package controllers

import (
	"github.com/streadway/amqp"
	"tg_sender/config"
	"tg_sender/core"
	"tg_sender/pkg/logger"
	"tg_sender/pkg/rabbit"
)

type ControllerMake struct {
	log logger.Logger
	cfg config.Config
	publisher *rabbit.Publisher
}

func New(log logger.Logger, cfg config.Config, conn *rabbit.Connection) ControllerMake {
	pub, err := rabbit.NewPublisher(conn, cfg.Exchange, log)
	if err != nil {
		log.Fatal(err.Error())
	}

	return ControllerMake{
		log: log,
		cfg: cfg,
		publisher: pub,
	}
}

type Message struct {
	Num      int `json:"num"`
	Priority int `json:"priority"`
}

func (c ControllerMake) ProcessMessages(m Message) error {
	data := core.GenerateMessages(m.Num, m.Priority)

	return c.pub(c.cfg.PublishingRoutingKey, data, m.Priority)
}

func (c ControllerMake) pub(routingKey string, data []string, p int) error {
	for _, v := range data {
		err := c.publisher.Push(c.cfg.PublishingRoutingKey, amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "plain/text",
			Body:         []byte(v),
			Priority: uint8(p),
		})
		if err != nil {
			c.log.Error(err.Error())

			return err
		}
	}
	return nil
}