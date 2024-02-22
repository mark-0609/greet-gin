package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"greet_gin/database"
	"greet_gin/models"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type RabbitMqController struct{}

// CreateExchange 声明信道
func (t RabbitMqController) CreateExchange(c *gin.Context) {
	var req database.ExchangeEntity
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, Response{
			Code: 0,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}
	rabbit := database.GetRabbitMqConn()
	if err := rabbit.DeclareExchange(req.Name, req.Type, req.Durable, req.AutoDelete, req.NoWait); err != nil {
		logrus.Errorf("Error while declare exchange:%v", err.Error())
		c.JSON(500, Response{
			Code: 0,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}
	c.JSON(200, Response{
		Code: 0,
		Msg:  "ok",
		Data: nil,
	})
	return
}

// CreateQueue 声明队列
func (t RabbitMqController) CreateQueue(c *gin.Context) {
	var req database.QueueEntity
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, Response{
			Code: 0,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}

	rabbit := database.GetRabbitMqConn()
	if err := rabbit.DeclareQueue(req.Name, req.Durable, req.AutoDelete, req.Exclusive, req.NoWait); err != nil {
		logrus.Errorf("Error while declare queue:%v", err.Error())
		c.JSON(500, Response{
			Code: 0,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}
	c.JSON(200, Response{
		Code: 0,
		Msg:  "ok",
		Data: nil,
	})
	return
}

type QueueBindReq struct {
	Queue    string `json:"queue"`
	Exchange string `json:"exchange"`
	NoWait   bool   `json:"nowait"`
	Keys     string `json:"keys"` // 英文逗号分割，绑定多个路由key
}

// BindQueue Exchange绑定队列
func (t RabbitMqController) BindQueue(c *gin.Context) {
	var entity QueueBindReq
	if err := c.ShouldBindJSON(&entity); err != nil {
		c.JSON(400, Response{
			Code: 0,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}

	var keys []string
	for _, k := range strings.Split(entity.Keys, ",") {
		keys = append(keys, k)
	}
	rabbit := database.GetRabbitMqConn()
	if err := rabbit.BindQueue(entity.Queue, entity.Exchange, keys, entity.NoWait); err != nil {
		logrus.Errorf("Error while bind queue:%v", err.Error())
		c.JSON(500, Response{
			Code: 0,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}
	c.JSON(200, Response{
		Code: 0,
		Msg:  "ok",
		Data: nil,
	})
	return
}

// ProductMq 生产数据
func (t RabbitMqController) ProductMq(c *gin.Context) {

	var entity database.MessageEntity
	if err := c.ShouldBindJSON(&entity); err != nil {
		c.JSON(400, Response{
			Code: 0,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				logrus.Errorf("publish message fail :%v ", err)
			}
		}()
		var articles []models.Article
		db := database.GetDb()
		err := db.Where("status=1").Limit(10).Select("id").Find(&articles).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logrus.Errorf("article no data err:%v", err.Error())
				return
			}
			logrus.Errorf("database err:%v", err.Error())
			return
		}
		rabbit := database.GetRabbitMqConn()
		err = rabbit.Channel.Confirm(false)
		if err != nil {
			logrus.Errorf("Failed to enable publisher confirms:%v", err.Error())
			return
		}
		confirms := rabbit.Channel.NotifyPublish(make(chan amqp.Confirmation, len(articles)))
		//defer rabbit.Close()
		for _, data := range articles {
			res, err := json.Marshal(data.Id)
			if err != nil {
				logrus.Errorf("json marshal err:%v", err.Error())
				return
			}
			// TODO: 需要开启confirm模式 手动ack
			if err = rabbit.Channel.Publish(entity.Exchange, entity.Key, false, false, amqp.Publishing{
				DeliveryMode:    amqp.Persistent,
				Headers:         amqp.Table{},
				ContentType:     "text/plain",
				ContentEncoding: "",
				Priority:        entity.Priority,
				Body:            res,
			}); err != nil {
				logrus.Errorf("database err:%v", err.Error())
				return
			}
		}
		select {
		case confirm := <-confirms:
			if !confirm.Ack {
				logrus.Errorf("Failed delivery of message with body...", confirm)
				// 可以在这里实现重试逻辑
				return
			}
		}
	}()

	c.JSON(200, Response{
		Code: 0,
		Msg:  "ok",
		Data: nil,
	})
	return
}

type ConsumeMqReq struct {
	QueueName        string `json:"queueName"`
	Consumer         string `json:"consumer"` // 指定消费者
	HasNewConnection bool   `json:"hasNewConnection"`
}

// ConsumeMq 消费
func (t RabbitMqController) ConsumeMq(c *gin.Context) {

	var req ConsumeMqReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, Response{
			Code: 0,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}
	redisClient := database.GetRedisClient()
	go database.ConsumeMessagesWithAck(req.QueueName, func(body amqp.Delivery) error {
		var data int
		err := json.Unmarshal(body.Body, &data)
		if err != nil {
			logrus.Errorf("err:%v", err)
			return err
		}
		// 避免重复消费 可以使用全局唯一字段判断，或者将消费的数据id存入redis
		result, err := redisClient.Get(strconv.Itoa(data)).Result()
		if !errors.Is(err, redis.Nil) {
			logrus.Infof("消息已被消费,忽略 %s", strconv.Itoa(data))
			// _ = body.Reject(false)
			return nil
		}
		if len(result) > 0 {
			logrus.Infof("redis data: %s", result)
		}
		if err := updateArticleData(data); err != nil {
			logrus.Errorf("updateArticleData err:%v", err)
			return err
		}
		return nil
	}, func() {})
	// go func() {
	// 	defer func() {
	// 		if err := recover(); err != nil {
	// 			logrus.Errorf("consume message fail :%v ", err)
	// 		}
	// 	}()
	// 	rabbit := database.GetRabbitMqConn()
	// 	if req.HasNewConnection {
	// 		rabbit = database.RabbitMqInit()
	// 	}
	// 	//TODO: 待关闭
	// 	defer func(rabbit *database.RabbitMQ) {
	// 		err := rabbit.Close()
	// 		logrus.Infof("rabbitmq consume close success")
	// 		if err != nil {
	// 			logrus.Errorf("rabbitmq consume close fail :%v ", err)
	// 		}
	// 	}(rabbit)
	// 	redisClient := database.GetRedisClient()
	// 	msg := make(chan []byte)
	// 	consumer := req.Consumer
	// 	err := rabbit.ConsumeQueue(req.QueueName, consumer, msg, false, func(body amqp.Delivery) (err error) {
	// 		var data int
	// 		err = json.Unmarshal(body.Body, &data)
	// 		if err != nil {
	// 			logrus.Errorf("err:%v", err)
	// 			return err
	// 		}
	// 		// 避免重复消费 可以使用全局唯一字段判断，或者将消费的数据id存入redis
	// 		result, err := redisClient.Get(strconv.Itoa(data)).Result()
	// 		if !errors.Is(err, redis.Nil) {
	// 			logrus.Infof("消息已被消费,忽略 %s", strconv.Itoa(data))
	// 			_ = body.Reject(false)
	// 			return
	// 		}
	// 		if len(result) > 0 {
	// 			logrus.Infof("redis data: %s", result)
	// 		}
	// 		if err := updateArticleData(data); err != nil {
	// 			logrus.Errorf("updateArticleData err:%v", err)
	// 			return err
	// 		}
	// 		return nil
	// 	})
	// 	if err != nil {
	// 		logrus.Errorf("[amqp] consume queue error: %s\n", err)
	// 		return
	// 	}
	// 	for {
	// 		<-msg
	// 	}
	// }()
	c.JSON(200, Response{
		Code: 0,
		Msg:  "ok",
		Data: nil,
	})
	return
}

func updateArticleData(id int) error {
	var article models.Article
	article.Status = 4
	db := database.GetDb()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(models.Article{}).Where("id= ?", id).Update(&article).Error; err != nil {
			logrus.Errorf("article update err:%v", err.Error())
			return err
		}
		redisClient := database.GetRedisClient()
		err := redisClient.Set(strconv.Itoa(id), id, time.Second*600).Err()
		if err != nil {
			logrus.Errorf("article redis set err:%v", err.Error())
			return err
		}
		return nil
	})
}

type DeadReq struct {
	HasCreateDeadQueue bool `json:"has_create_dead_queue"`
	Publish            bool `json:"publish"`
	DeadConsume        bool `json:"dead_consume"`
}

func (r RabbitMqController) Dead(c *gin.Context) {
	var req DeadReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, Response{
			Code: 0,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}
	if req.HasCreateDeadQueue {
		createDeadQueue()
		c.JSON(200, Response{
			Code: 0,
			Msg:  "createDeadQueue ok",
			Data: nil,
		})
		return
	}

	if req.Publish {
		publish()
		c.JSON(200, Response{
			Code: 0,
			Msg:  "publish ok",
			Data: nil,
		})
		return
	}
	if req.DeadConsume {
		deadConsume()
		return
	}
	consume()

	c.JSON(200, Response{
		Code: 0,
		Msg:  "ok",
		Data: nil,
	})
	return
}

var (
	normalQueue      = "normal_queue"
	normalExchange   = "normal_exchange"
	normalRoutingKey = "normal_exchange_routing_key"
	deadQueue        = "dead_queue"
	deadExchange     = "dead_exchange"
	deadRoutingKey   = "dead_exchange_routing_key"
)

// 普通队列生产者
func publish() {
	message := "msg" + strconv.Itoa(int(time.Now().Unix()))
	fmt.Println(message)

	// 发布消息
	rabbitMqConn := database.GetRabbitMqConn()
	err := rabbitMqConn.Channel.Publish(normalExchange, normalRoutingKey, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(message),
	})
	if err != nil {
		logrus.Errorf("Error publishing Rabbit:%v", err)
		return
	}
}

// 普通队列消费
func consume() {
	rabbitMqConn := database.GetRabbitMqConn()
	msgsCh, err := rabbitMqConn.Channel.Consume(normalQueue, "", false, false, false, false, nil)
	if err != nil {
		logrus.Errorf("消费失败:%v", err)
		return
	}

	forever := make(chan bool)
	go func() {
		for d := range msgsCh {
			// 要实现的逻辑
			logrus.Infof("接收的消息: %s", d.Body)

			// 手动应答
			d.Ack(false)
			//d.Reject(true)
		}
	}()
	logrus.Infof("[*] Waiting for message, To exit press CTRL+C")
	for {
		<-forever
	}
}

// 死信队列消费
func deadConsume() {
	rabbitMqConn := database.GetRabbitMqConn()
	msgsCh, err := rabbitMqConn.Channel.Consume(deadQueue, "", false, false, false, false, nil)
	if err != nil {
		logrus.Errorf("err:%v", err)
		return
	}

	forever := make(chan bool)
	go func() {
		for d := range msgsCh {
			// 要实现的逻辑
			logrus.Infof("接收的消息: %s", d.Body)

			// 手动应答
			d.Ack(false)
			//d.Reject(true)
		}
	}()
	logrus.Infof("[*] Waiting for message, To exit press CTRL+C")
	for {
		<-forever
	}
}

// 声明死信exchange\路由key、队列
func createDeadQueue() {
	rabbitMqConn := database.GetRabbitMqConn()

	_, err := rabbitMqConn.Channel.QueueDeclare(normalQueue, true, false, false, false, amqp.Table{
		"x-message-ttl":             5000,           // 消息过期时间,毫秒
		"x-dead-letter-exchange":    deadExchange,   // 指定死信交换机
		"x-dead-letter-routing-key": deadRoutingKey, // 指定死信routing-key
	})
	if err != nil {
		logrus.Errorf("创建normal队列失败：%v", err)
		return
	}

	err = rabbitMqConn.Channel.ExchangeDeclare(normalExchange, amqp.ExchangeDirect, true, false, false, false, nil)
	if err != nil {
		logrus.Errorf("创建normal交换机失败：%v", err)
		return
	}

	err = rabbitMqConn.Channel.QueueBind(normalQueue, normalRoutingKey, normalExchange, false, nil)
	if err != nil {
		logrus.Errorf("normal：队列、交换机、routing-key 绑定失败 :%v", err)
		return
	}

	// 声明死信队列
	// args 为 nil。切记不要给死信队列设置消息过期时间,否则失效的消息进入死信队列后会再次过期。
	_, err = rabbitMqConn.Channel.QueueDeclare(deadQueue, true, false, false, false, nil)
	if err != nil {
		logrus.Errorf("创建dead队列失败:%v", err)
		return
	}

	// 声明交换机
	err = rabbitMqConn.Channel.ExchangeDeclare(deadExchange, amqp.ExchangeDirect, true, false, false, false, nil)
	if err != nil {
		logrus.Errorf("创建dead交换机失败:%v", err)
		return
	}

	// 队列绑定（将队列、routing-key、交换机三者绑定到一起）
	err = rabbitMqConn.Channel.QueueBind(deadQueue, deadRoutingKey, deadExchange, false, nil)
	if err != nil {
		logrus.Errorf("dead：队列、交换机、routing-key 绑定失败:%v", err)
		return
	}
}
