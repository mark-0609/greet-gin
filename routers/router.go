package routers

import (
	"fmt"
	"greet_gin/controllers"

	"strings"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {

	gin.ForceConsoleColor()

	router := gin.New()
	router.Use(gin.Logger(), gin.CustomRecovery(func(c *gin.Context, err interface{}) {
		//middleware.PanicInfo(c, err)
	}))

	web := router.Group("/web")
	TestController := new(controllers.TestController)
	web.GET("/test/test", TestController.Test)
	web.GET("/test/es", TestController.Es)
	web.GET("/test/createQueue", TestController.CreateQueue)
	web.GET("/test/createExchange", TestController.CreateExchange)

	searchController := new(controllers.SearchController)
	web.POST("/search/index", searchController.Index)

	return router
}

func Cors() gin.HandlerFunc {

	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")
		var headerKeys []string
		for k, _ := range c.Request.Header {
			headerKeys = append(headerKeys, k)
		}
		headerStr := strings.Join(headerKeys, ", ")
		if headerStr != "" {
			headerStr = fmt.Sprintf("access-control-allow-origin, access-control-allow-headers, %s", headerStr)
		} else {
			headerStr = "access-control-allow-origin, access-control-allow-headers"
		}
		if origin != "" {
			c.Header("X-Content-Type-Options", "nosniff")
			//https://developer.mozilla.org/zh-CN/docs/Web/HTTP/X-Frame-Options
			c.Header("X-Frame-Options", "DENY") // 、SAMEORIGIN、ALLOW-FROM origin
			// c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			// c.Header("Access-Control-Allow-Origin", "*") // 这是允许访问所有域
			// c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,UPDATE") //服务器支持的所有跨域请求的方法,为了避免浏览次请求的多次'预检'请求
			//  header的类型
			// c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Token,session,X_Requested_With,Accept, Origin, Host, Connection, Accept-Encoding, Accept-Language,DNT, X-CustomHeader, Keep-Alive, User-Agent, X-Requested-With, If-Modified-Since, Cache-Control, Content-Type, Pragma")
			//              允许跨域设置                                                                                                      可以返回其他子段
			// c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers,Cache-Control,Content-Language,Content-Type,Expires,Last-Modified,Pragma,FooBar") // 跨域关键设置 让浏览器可以解析
			// c.Header("Access-Control-Max-Age", "172800") // 缓存请求信息 单位为秒
			// c.Header("Access-Control-Allow-Credentials", "false")                                                                                                                                                  //  跨域请求是否需要带cookie信息 默认设置为true
			c.Header("content-type", "application/json;charset=utf-8") // 设置返回格式是json
		}

		//放行所有OPTIONS方法
		//if method == "OPTIONS" {
		//    c.JSON(http.StatusOK, "Options Request!")
		//}
		if method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		// 处理请求
		c.Next() //  处理请求
	}
}
