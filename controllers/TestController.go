package controllers

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"greet_gin/database"
	"greet_gin/models"
	"greet_gin/service/user"
)

type TestController struct {
	Controller
}

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
			c.JSON(500, t.FailMsg(err.Error()))
			return
		}
		logrus.Errorf("database err:%v", err.Error())
		c.JSON(500, t.FailMsg(err.Error()))
		return
	}
	userEs, err := user.NewUserService(context.Background())
	if err != nil {
		logrus.Errorf("err:%v", err)
		c.JSON(500, t.FailMsg(err.Error()))
		return
	}
	err = userEs.BatchAdd(userModel)
	if err != nil {
		logrus.Errorf("BatchAdd err:%v", err)
		c.JSON(500, t.FailMsg(err.Error()))
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
	logrus.Infof("DATA:%v", len(result))
	c.JSON(200, t.DataMsg(result))
	return
}
