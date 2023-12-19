package controllers

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"greet_gin/database"
	"greet_gin/models"
	"strconv"
	"strings"
	"time"
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
		err := db.Where("status=1").Limit(100000).Select("id").Find(&articles).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logrus.Errorf("article no data err:%v", err.Error())
				return
			}
			logrus.Errorf("database err:%v", err.Error())
			return
		}
		rabbit := database.GetRabbitMqConn()
		//defer rabbit.Close()
		for _, data := range articles {
			res, err := json.Marshal(data.Id)
			if err != nil {
				logrus.Errorf("json marshal err:%v", err.Error())
				return
			}
			if err = rabbit.Publish(entity.Exchange, entity.Key, entity.DeliveryMode, entity.Priority, string(res)); err != nil {
				logrus.Errorf("database err:%v", err.Error())
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
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logrus.Errorf("consume message fail :%v ", err)
			}
		}()
		rabbit := database.GetRabbitMqConn()
		if req.HasNewConnection {
			rabbit = database.RabbitMqInit()
		}
		//TODO: 待关闭
		defer func(rabbit *database.RabbitMQ) {
			err := rabbit.Close()
			logrus.Infof("rabbitmq consume close success")
			if err != nil {
				logrus.Errorf("rabbitmq consume close fail :%v ", err)
			}
		}(rabbit)
		redisClient := database.GetRedisClient()
		msg := make(chan []byte)
		consumer := req.Consumer
		err := rabbit.ConsumeQueue(req.QueueName, consumer, msg, false, func(body amqp.Delivery) (err error) {
			var data int
			err = json.Unmarshal(body.Body, &data)
			if err != nil {
				logrus.Errorf("err:%v", err)
				return err
			}
			// 避免重复消费 可以使用全局唯一字段判断，或者将消费的数据id存入redis
			result, err := redisClient.Get(strconv.Itoa(data)).Result()
			if !errors.Is(err, redis.Nil) {
				logrus.Infof("消息已被消费,忽略 %s", strconv.Itoa(data))
				_ = body.Reject(false)
				return
			}
			if len(result) > 0 {
				logrus.Infof("redis data: %s", result)
			}
			if err := updateArticleData(data); err != nil {
				logrus.Errorf("updateArticleData err:%v", err)
				return err
			}
			return nil
		})
		if err != nil {
			logrus.Errorf("[amqp] consume queue error: %s\n", err)
			return
		}
		for {
			<-msg
		}
	}()
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
