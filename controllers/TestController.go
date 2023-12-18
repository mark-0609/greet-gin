package controllers

import (
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
