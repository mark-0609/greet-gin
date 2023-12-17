package controllers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"greet_gin/database"
	"greet_gin/models"
	"greet_gin/service"
)

type TestController struct{}

func (t TestController) Test(c *gin.Context) {

	//userEs := service.UserEsInit(database.InitES(), c.Request.Context())
	//err = userEs.Search(c, userModel)
	//if err != nil {
	//	logrus.Errorf("err:%v", err)
	//	c.JSON(200, err)
	//	return
	//}

	c.JSON(200, Response{
		Code: 0,
		Msg:  "test",
		Data: nil,
	})
	return
}

func (t TestController) Es(c *gin.Context) {

	var userModel []models.User
	db := database.GetDb()
	err := db.Find(&userModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			FailMsg("no data")
			return
		}
		logrus.Errorf("database err:%v", err.Error())
		FailMsg(err.Error())
		return
	}

	userEs := service.UserEsInit(database.InitES(), c.Request.Context())
	err = userEs.BatchAdd(c, userModel)
	if err != nil {
		logrus.Errorf("err:%v", err)
		c.JSON(200, err)
		return
	}
	type Res struct {
		Id   int
		Name string
	}

	var result []Res
	for _, r := range userModel {
		var res Res
		res.Name = r.UserName
		res.Id = r.Id
		result = append(result, res)
	}

	c.JSON(200, DataMsg(result))
	return
}

// CreateExchange 声明信道
func (t TestController) CreateExchange(c *gin.Context) {
	var req database.ExchangeEntity
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, Response{
			Code: 0,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}
	rabbit := new(database.RabbitMQ)
	if err := rabbit.Connect(); err != nil {
		c.JSON(500, Response{
			Code: 0,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}
	defer rabbit.Close()

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
func (t TestController) CreateQueue(c *gin.Context) {
	var req database.QueueEntity
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, Response{
			Code: 0,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}

	rabbit := new(database.RabbitMQ)
	if err := rabbit.Connect(); err != nil {
		c.JSON(500, Response{
			Code: 0,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}
	defer rabbit.Close()
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
func (t TestController) BindQueue(c *gin.Context) {
	var entity QueueBindReq
	if err := c.ShouldBindJSON(&entity); err != nil {
		c.JSON(400, Response{
			Code: 0,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}

	rabbit := new(database.RabbitMQ)
	if err := rabbit.Connect(); err != nil {
		c.JSON(500, Response{
			Code: 0,
			Msg:  err.Error(),
			Data: nil,
		})
		return
	}
	defer rabbit.Close()

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
func (t TestController) ProductMq(c *gin.Context) {

	var articles []models.Article
	db := database.GetDb()
	err := db.Where("status=1").Limit(100000).Find(&articles).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			FailMsg("no data")
			return
		}
		logrus.Errorf("database err:%v", err.Error())
		FailMsg(err.Error())
		return
	}

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
		rabbit := new(database.RabbitMQ)
		defer rabbit.Close()
		if err = rabbit.Connect(); err != nil {
			c.JSON(500, Response{
				Code: 0,
				Msg:  err.Error(),
				Data: nil,
			})
			return
		}

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

	//
	c.JSON(200, Response{
		Code: 0,
		Msg:  "ok",
		Data: nil,
	})
	return
}

// ConsumeMq 消费
func (t TestController) ConsumeMq(c *gin.Context) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				logrus.Errorf("consume message fail :%v ", err)
			}
		}()
		rabbit := new(database.RabbitMQ)
		if err := rabbit.Connect(); err != nil {
			c.JSON(500, Response{
				Code: 0,
				Msg:  err.Error(),
				Data: nil,
			})
			return
		}
		defer rabbit.Close()

		message := make(chan []byte)

		if err := rabbit.ConsumeQueue("queue-1", message); err != nil {
			c.JSON(500, Response{
				Code: 0,
				Msg:  err.Error(),
				Data: nil,
			})
			return
		}
		logrus.Infof("Received message %s\n", <-message)
	}()

}
