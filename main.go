package main

import (
	"fmt"
	"greet_gin/config"
	"greet_gin/database"
	"greet_gin/routers"
)

func main() {

	config.Setup()

	database.Init()

	database.InitES()
	//fmt.Println("es client", database.GetElasticClient())

	database.RabbitMqInit()
	fmt.Println("rabbitMq init successful:", database.GetRabbitMqConn())

	r := routers.InitRouter()
	r.Run(":" + config.ServerSetting.Port)
}
