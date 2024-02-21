package database

import (
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

var (
	rabbitMQURL = "amqp://guest:guest@114.132.210.241:5672/"
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
var connRetry = 3

// RabbitMqInit 初始化
func RabbitMqInit() *RabbitMQ {
	rabbitMqConn = new(RabbitMQ)
	if err := rabbitMqConn.Connect(); err != nil {
		logrus.Errorf("Error connecting to RabbitMQ：%v", err)
		return nil
	}
	return rabbitMqConn
}

func failOnError(err error, msg string) {
	if err != nil {
		logrus.Errorf("%s: `%s`", msg, err)
	}
}

// CreateConnAndChannel 创建新的连接和管道
func CreateConnAndChannel() (*amqp.Channel, *amqp.Connection) {
	conn, err := amqp.Dial(rabbitMQURL)
	failOnError(err, "Failed to connect to RabbitMQ")
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	return ch, conn
}

// ConsumeMessagesWithAck 实时消费某个队列任务
func ConsumeMessagesWithAck(queueName string, processMessage func(amqp.Delivery) error, errorFunc func()) {
	ch, _ := CreateConnAndChannel()
	for {
		msgs, err := ch.Consume(
			queueName,
			"",
			false,
			false,
			false,
			false,
			nil,
		)
		failOnError(err, "Failed to register a consumer")
		for d := range msgs {
			logrus.Infof("Received a message: %s", d.Body)
			// time.Sleep(time.Second * 3)
			// 在这里处理消息，确保没有发生错误，否则消息可能会被丢失
			err := processMessage(d)
			logrus.Errorf("processing message: %v", err)
			if err == nil {
				d.Ack(false) // 确认消息已经被处理
			} else {
				logrus.Errorf("Error processing message: %v", err)
				// 可以在这里实现错误处理和重试逻辑
				// 处理消息失败时，重新发布消息到队列
				errorFunc()
				d.Nack(false, true)
			}
		}
	}
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
	r.Conn, err = amqp.Dial(rabbitMQURL)
	if err != nil {
		logrus.Errorf("[amqp] connect error: %s\n", err)
		for i := 0; i <= connRetry; i++ {
			r.Conn, err = amqp.Dial(rabbitMQURL)
			if err != nil {
				logrus.Errorf("[amqp] connect retry count :%v, error: %s\n", i, err)
			} else {
				break
			}
		}
		if err != nil {
			return err
		}
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
