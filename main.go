package main

import (
	"context"
	"fmt"
	_ "github.com/mkevac/debugcharts"
	"github.com/sirupsen/logrus"
	"greet_gin/config"
	"greet_gin/database"
	"greet_gin/routers"
	"net/http"
	_ "net/http/pprof"
)

func main() {

	go func() {
		err := http.ListenAndServe(":10108", nil)
		if err != nil {
			logrus.Error("Error listening")
		}
	}()
	config.Setup()

	database.Init()

	database.InitES()
	fmt.Println("es client", database.GetElasticClient(context.Background()))

	database.RabbitMqInit()
	fmt.Println("rabbitMq init successful:", database.GetRabbitMqConn())

	r := routers.InitRouter()
	r.Run(":" + config.ServerSetting.Port)

}
