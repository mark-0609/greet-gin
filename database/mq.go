package database

import (
	"flag"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

var (
	amqpUri = flag.String("amqp", "amqp://guest:guest@114.132.210.241:5672/", "amqp uri")
)

// Entity for HTTP Request Body: Message/Exchange/Queue/QueueBind JSON Input
type MessageEntity struct {
	Exchange     string `json:"exchange"`
	Key          string `json:"key"`
	DeliveryMode uint8  `json:"deliverymode"`
	Priority     uint8  `json:"priority"`
	Body         string `json:"body"`
}

type ExchangeEntity struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Durable    bool   `json:"durable"`
	AutoDelete bool   `json:"autodelete"`
	NoWait     bool   `json:"nowait"`
}

type QueueEntity struct {
	Name       string `json:"name"`
	Durable    bool   `json:"durable"`
	AutoDelete bool   `json:"autodelete"`
	Exclusive  bool   `json:"exclusive"`
	NoWait     bool   `json:"nowait"`
}

type QueueBindEntity struct {
	Queue    string   `json:"queue"`
	Exchange string   `json:"exchange"`
	NoWait   bool     `json:"nowait"`
	Keys     []string `json:"keys"` // bind/routing keys
}

// RabbitMQ Operate Wrapper
type RabbitMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
	Done    chan error
}

var rabbitMqConn *RabbitMQ

// RabbitMqInit 初始化
func RabbitMqInit() *RabbitMQ {
	rabbitMqConn = new(RabbitMQ)
	if err := rabbitMqConn.Connect(); err != nil {
		logrus.Errorf("Error connecting to RabbitMQ：%v", err)
		return nil
	}
	return rabbitMqConn
}

// GetRabbitMqConn 获取rabbitmq连接
func GetRabbitMqConn() *RabbitMQ {
	logrus.Infof("GetRabbitMqConn:%v", rabbitMqConn)
	if rabbitMqConn.Conn.IsClosed() {
		return RabbitMqInit()
	}
	return rabbitMqConn
}

func (r *RabbitMQ) Connect() (err error) {
	r.Conn, err = amqp.Dial(*amqpUri)
	if err != nil {
		logrus.Errorf("[amqp] connect error: %s\n", err)
		return err
	}
	r.Channel, err = r.Conn.Channel()
	if err != nil {
		logrus.Errorf("[amqp] get Channel error: %s\n", err)
		return err
	}
	r.Done = make(chan error)
	return nil
}

// DeclareQueue 1初始化队列
func (r *RabbitMQ) DeclareQueue(name string, durable, autodelete, exclusive, nowait bool) (err error) {
	_, err = r.Channel.QueueDeclare(name, durable, autodelete, exclusive, nowait, nil)
	if err != nil {
		logrus.Errorf("[amqp] declare queue error: %s\n", err)
		return err
	}
	return nil
}

// DeclareExchange 2初始化信道
func (r *RabbitMQ) DeclareExchange(name, typ string, durable, autodelete, nowait bool) (err error) {
	err = r.Channel.ExchangeDeclare(name, typ, durable, autodelete, false, nowait, nil)
	if err != nil {
		logrus.Errorf("[amqp] declare exchange error: %s\n", err)
		return err
	}
	return nil
}

// BindQueue 3绑定队列
func (r *RabbitMQ) BindQueue(queue, exchange string, keys []string, nowait bool) (err error) {
	for _, key := range keys {
		if err = r.Channel.QueueBind(queue, key, exchange, nowait, nil); err != nil {
			logrus.Errorf("[amqp] bind queue error: %s\n", err)
			return err
		}
	}
	return nil
}

// Publish 4生产队列数据
func (r *RabbitMQ) Publish(exchange, key string, deliverymode, priority uint8, body string) (err error) {
	err = r.Channel.Publish(exchange, key, false, false,
		amqp.Publishing{
			Headers:         amqp.Table{},
			ContentType:     "text/plain",
			ContentEncoding: "",
			DeliveryMode:    deliverymode,
			Priority:        priority,
			Body:            []byte(body),
		},
	)
	if err != nil {
		logrus.Errorf("[amqp] publish message error: %s\n", err)
		return err
	}
	return nil
}

// ConsumeQueue 5消费某个队列
func (r *RabbitMQ) ConsumeQueue(queue, consumer string, message chan []byte, autoAck bool, fn func(data amqp.Delivery) error) (err error) {
	deliveries, err := r.Channel.Consume(queue, consumer, autoAck, false, false, false, nil)
	if err != nil {
		logrus.Errorf("[amqp] consume queue error: %s\n", err)
		return err
	}
	go func(deliveries <-chan amqp.Delivery, done chan error, message chan []byte) {
		for d := range deliveries {
			message <- d.Body
			if err := fn(d); err != nil {
				return
			}
			if err = d.Ack(autoAck); err != nil {
				logrus.Errorf("queue:%v, ack error :%v", queue, err)
				return
			}
		}
		done <- nil
	}(deliveries, r.Done, message)
	return nil
}

// DeleteExchange 删除信道
func (r *RabbitMQ) DeleteExchange(name string) (err error) {
	err = r.Channel.ExchangeDelete(name, false, false)
	if err != nil {
		logrus.Errorf("[amqp] delete exchange error: %s\n", err)
		return err
	}
	return nil
}

// DeleteQueue 删除队列
func (r *RabbitMQ) DeleteQueue(name string) (err error) {
	// TODO: other property wrapper
	_, err = r.Channel.QueueDelete(name, false, false, false)
	if err != nil {
		logrus.Errorf("[amqp] delete queue error: %s\n", err)
		return err
	}
	return nil
}

// UnBindQueue 解除绑定队列
func (r *RabbitMQ) UnBindQueue(queue, exchange string, keys []string) (err error) {
	for _, key := range keys {
		if err = r.Channel.QueueUnbind(queue, key, exchange, nil); err != nil {
			logrus.Errorf("[amqp] unbind queue error: %s\n", err)
			return err
		}
	}
	return nil
}

// Close 关闭rabbitMq
func (r *RabbitMQ) Close() (err error) {
	err = r.Conn.Close()
	if err != nil {
		logrus.Errorf("[amqp] close error: %s\n", err)
		return err
	}
	return nil
}
