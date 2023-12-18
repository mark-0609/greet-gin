package controllers

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"greet_gin/database"
	"greet_gin/models"
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
	Keys     string `json:"keys"` // bind/routing keys
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

	rabbit := database.GetRabbitMqConn()
	if err := rabbit.BindQueue(entity.Queue, entity.Exchange, []string{entity.Keys}, entity.NoWait); err != nil {
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
		err := db.Where("status=1").Limit(100000).Select("id,article_name,content").Find(&articles).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				FailMsg("no data")
				return
			}
			logrus.Errorf("database err:%v", err.Error())
			FailMsg(err.Error())
			return
		}
		rabbit := database.GetRabbitMqConn()
		for _, data := range articles {
			res, _ := json.Marshal(data)
			if err = rabbit.Publish(entity.Exchange, entity.Key, entity.DeliveryMode, entity.Priority, string(res)); err != nil {
				c.JSON(500, Response{
					Code: 0,
					Msg:  err.Error(),
					Data: nil,
				})
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
	QueueName string `json:"queueName"`
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

		//message := make(chan []byte)
		//rabbit := database.GetRabbitMqConn()
		//if err := rabbit.ConsumeQueue(req.QueueName, message); err != nil {
		//	c.JSON(500, Response{
		//		Code: 0,
		//		Msg:  err.Error(),
		//		Data: nil,
		//	})
		//	return
		//}
		//for {
		//	logrus.Infof("Received message %s\n", <-message)
		//}
		message := make(chan []byte)
		rabbit := database.GetRabbitMqConn()
		if err := rabbit.ConsumeQueue(req.QueueName, message, false); err != nil {
			c.JSON(500, Response{
				Code: 0,
				Msg:  err.Error(),
				Data: nil,
			})
			return
		}
		for {
			logrus.Infof("Received message %s\n", <-message)
		}
	}()
	c.JSON(200, Response{
		Code: 0,
		Msg:  "ok",
		Data: nil,
	})
	return
}
