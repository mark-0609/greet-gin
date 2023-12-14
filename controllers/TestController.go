package controllers

import (
	"greet_gin/database"
	"greet_gin/models"
	"greet_gin/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type TestController struct{}

func (t TestController) Test(c *gin.Context) {
	userEs := service.NewUserES(database.InitES())
	var userModel []models.User
	db := database.GetDb()
	db.Find(&userModel)
	err := userEs.BatchAdd(c, userModel)
	if err != nil {
		logrus.Errorf("err:%v", err)
		c.JSON(200, err)
		return
	}
}
