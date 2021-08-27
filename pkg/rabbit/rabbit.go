package rabbit

import (
	"context"
	"fmt"
	"github.com/streadway/amqp"
	"sync/atomic"
	"tg_sender/pkg/logger"
	"time"
)

type Connection struct {
	*amqp.Connection
	url            string
	log            logger.Logger
	reconnectDelay time.Duration
	try            int
}

// Dial wrap amqp.Dial, dial and get a reconnect connection
func Dial(url string, log logger.Logger, reconnect time.Duration) (*Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	c := &Connection{conn, url, log, reconnect, 0}

	go func() {
		for {
			reason, ok := <-c.Connection.NotifyClose(make(chan *amqp.Error))
			// exit this goroutine if closed by developer
			if !ok {
				c.log.Debug("Connection closed")
				break
			}
			c.log.Debug(fmt.Sprintf("connection closed, reason: %v", reason))

			// reconnect if not closed by developer
			for {
				// wait 1s for reconnect
				time.Sleep(c.reconnectDelay * time.Second * time.Duration(c.try))

				conn, err := amqp.Dial(c.url)
				if err == nil {
					c.Connection = conn
					c.log.Info(fmt.Sprintf("AMQP Reconnected"))
					break
				}

				c.log.Info(fmt.Sprintf("Reconnect try %v failed", c.try))
				c.try++
			}
		}
	}()

	return c, nil
}

// Channel wrap amqp.Connection.Channel, get a auto reconnect channel
func (c *Connection) Channel() (*Channel, error) {
	ch, err := c.Connection.Channel()
	if err != nil {
		return nil, err
	}

	channel := &Channel{
		Channel:    ch,
		connection: c,
	}

	go func() {
		for {
			reason, ok := <-channel.Channel.NotifyClose(make(chan *amqp.Error))
			// exit this goroutine if closed by developer
			if !ok || channel.IsClosed() {
				c.log.Debug(fmt.Sprintf("channel closed"))
				if err := channel.Close(); err != nil {
					c.log.Error(fmt.Sprintf("channel close err: %v", err.Error()))
				} // close again, ensure closed flag set when connection closed
				break
			}
			c.log.Debug(fmt.Sprintf("channel closed, reason: %v", reason))

			// reconnect if not closed by developer
			for {
				// wait 1s for connection reconnect
				time.Sleep(c.reconnectDelay * time.Second)

				ch, err := c.Connection.Channel()
				if err == nil {
					c.log.Debug("channel recreate success")
					channel.Channel = ch
					break
				}
				c.log.Debug(fmt.Sprintf("channel recreate failed, err: %v", err.Error()))
			}
		}
	}()

	return channel, nil
}

// Channel amqp.Channel wrapper
type Channel struct {
	*amqp.Channel
	closed     int32
	connection *Connection
}

// IsClosed indicate closed by developer
func (ch *Channel) IsClosed() bool {
	return atomic.LoadInt32(&ch.closed) == 1
}

// Close ensure closed flag set
func (ch *Channel) Close() error {
	if ch.IsClosed() {
		return amqp.ErrClosed
	}
	atomic.StoreInt32(&ch.closed, 1)

	return ch.Channel.Close()
}


// Consume wrap amqp.Channel.Consume, the returned delivery will end only when channel closed by developer
func (ch *Channel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	deliveries := make(chan amqp.Delivery)

	go func() {
		for {
			d, err := ch.Channel.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
			if err != nil {
				time.Sleep(ch.connection.reconnectDelay * time.Second)
				continue
			}

			for msg := range d {
				deliveries <- msg
			}

			// sleep before IsClose call. closed flag may not set before sleep.
			time.Sleep(2 * time.Second)

			if ch.IsClosed() {
				break
			}
		}
	}()

	return deliveries, nil
}

func (ch *Channel) Publish(exchange, routingKey string, mandatory, intermediate bool, msg amqp.Publishing) error {
	return ch.Channel.Publish(exchange, routingKey, mandatory, intermediate, msg)
}

func declareQueue(ch *Channel, name string) (amqp.Queue, error) {
	args := make(amqp.Table)
	args["x-max-priority"] = int64(9)
	return ch.Channel.QueueDeclare(
		name,  // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		args,   // arguments
	)
}

func declareExchange(ch *Channel, name string) error {
	args := make(amqp.Table)
	args["x-max-priority"] = int64(9)
	return ch.Channel.ExchangeDeclare(
		name,               // name
		amqp.ExchangeTopic, // type
		true,               // durable
		false,              // auto-deleted
		false,              // worker
		false,              // no-wait
		args,                // arguments
	)
}

// Consumer ...
type Consumer struct {
	conn         *Connection
	channel      *Channel
	exchangeName string
	routingKey   string
	queueName    string
	middlewares  []func(amqp.Delivery)
	messages     <-chan amqp.Delivery
	log          logger.Logger
	stopped      bool
}

// NewConsumer ...
func NewConsumer(conn *Connection, exchangeName, routingKey, queueName string, log logger.Logger) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		log.Error(fmt.Sprintf("Failed to connect %v", err.Error()))
		return nil, err
	}

	if err = declareExchange(ch, exchangeName); err != nil {
		log.Error(fmt.Sprintf("Exchange Declare: %v", err.Error()))
		return nil, err
	}
	if _, err := declareQueue(ch, queueName); err != nil {
		return nil, err
	}

	err = ch.QueueBind(queueName,
		routingKey,
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to bind Queue %v", err.Error()))
		return nil, err
	}

	return &Consumer{
		conn:         conn,
		channel:      ch,
		exchangeName: exchangeName,
		routingKey:   routingKey,
		queueName:    queueName,
		middlewares:  make([]func(amqp.Delivery), 0),
		log:          log,
	}, nil
}

// Run ...
func (c *Consumer) Run(ctx context.Context, consumerName string, handler func(amqp.Delivery)) error {
	var err error

	c.channel.Qos(1, 0, false)

	c.messages, err = c.channel.Consume(
		c.queueName,
		consumerName,
		false,
		false,
		false,
		false,
		nil)
	if err != nil {
		return err
	}

	for {
		select {
		case msg := <-c.messages:
			for _, middleware := range c.middlewares {
				middleware(msg)
			}
			c.log.Debug(string(msg.Body))
			handler(msg)
		case <-ctx.Done():
			{
				return nil
			}
		}
	}
}

func (c *Consumer) Close() error {
	return c.channel.Close()
}

// AppendMiddleware ...
func (c *Consumer) AppendMiddleware(f func(amqp.Delivery)) {
	c.middlewares = append(c.middlewares, f)
}

// Publisher ...
type Publisher struct {
	log          logger.Logger
	conn         *Connection
	channel      *Channel
	exchangeName string
}

// NewPublisher ...
func NewPublisher(conn *Connection, exchangeName string, log logger.Logger) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	err = declareExchange(ch, exchangeName)
	if err != nil {
		log.Error(fmt.Sprintf("Exchange Declare: %s", err.Error()))
		return nil, err
	}
	return &Publisher{
		conn:         conn,
		channel:      ch,
		exchangeName: exchangeName,
	}, nil
}

// Push ...
func (p *Publisher) Push(routingKey string, msg amqp.Publishing) error {
	err := p.channel.Publish(
		p.exchangeName,
		routingKey,
		false,
		false,
		msg,
	)
	return err
}

// Close ...
func (p *Publisher) Close() error {
	return p.channel.Close()
}