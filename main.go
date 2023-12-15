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
	fmt.Println("es client", database.GetElasticClient())

	r := routers.InitRouter()

	//r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Run("3001")
}
