package controllers

import (
	"github.com/gin-gonic/gin"
	"greet_gin/service/user"
)

type SearchController struct {
	Controller
}

//type SearchReq struct {
//	DataType string                      `json:"data_type"` // 查询的数据模型 article chat job
//	Query    map[interface{}]interface{} `json:"query"`     // 查询模型字段
//	Page     int                         `json:"page"`      // 页码
//	PageSize int                         `json:"page_size"` //
//}

func (s SearchController) Index(c *gin.Context) {

	var req user.UserSearchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, s.FailMsg(err.Error()))
		return
	}

	//es, _ := user.NewUserService(context.Background())
	//es.BatchAdd()

	c.JSON(200, s.SuccessMsg("result"))
	return
}
