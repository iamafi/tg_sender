package workers

import (
	"context"
	"fmt"
	"github.com/streadway/amqp"
	"tg_sender/config"
	"tg_sender/controllers"
	"tg_sender/pkg/logger"
	"tg_sender/pkg/rabbit"
	"tg_sender/service"
	"time"
)

type Subscriber struct {
	consumer *rabbit.Consumer
	log      logger.Logger
	cfg      config.Config
	ctrl     controllers.ControllerSend
	bot      service.BotService
}


func New(c *rabbit.Consumer, l logger.Logger, cfg config.Config, bot service.BotService) *Subscriber {
	return &Subscriber{
		consumer: c,
		log:      l,
		cfg:      cfg,
		ctrl:     controllers.Init(l, bot),
	}
}

func (s *Subscriber) handle(e amqp.Delivery) {

	msg := string(e.Body)
	err := s.ctrl.SendMessages(msg)
	if err != nil {
		s.log.Error("can't send a message to tg")
	}


	if err := e.Ack(false); err != nil {
		s.log.Error(fmt.Sprintf("On Ack error: %v", err.Error()))

		return
	}
	time.Sleep(time.Second)
}

func (s *Subscriber) Run(ctx context.Context) error {
	s.log.Info("Worker started")

	return s.consumer.Run(ctx, s.cfg.WorkerRoutingKey, s.handle)
}
