package main

import (
	"context"
	"fmt"
	"greet_gin/config"
	"greet_gin/database"
	"greet_gin/routers"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	_ "github.com/mkevac/debugcharts"
	"github.com/sirupsen/logrus"
)

func dummyCPUUsage() {
	var a uint64
	var t = time.Now()
	for {
		t = time.Now()
		a += uint64(t.Unix())
	}
}

func dummyAllocations() {
	var d []uint64

	for {
		for i := 0; i < 2*1024*1024; i++ {
			d = append(d, 42)
		}
		time.Sleep(time.Second * 10)
		fmt.Println(len(d))
		d = make([]uint64, 0)
		runtime.GC()
		time.Sleep(time.Second * 10)
	}
}

func main() {

	go dummyAllocations()
	go dummyCPUUsage()
	go func() {
		logrus.Info(http.ListenAndServe(":10108", nil))
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
