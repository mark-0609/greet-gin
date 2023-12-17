package controllers

import (
	"github.com/gin-gonic/gin"
)

type SearchController struct{}

type SearchReq struct {
	DataType string                      `json:"data_type"` // 查询的数据模型 article chat job
	Query    map[interface{}]interface{} `json:"query"`     // 查询模型字段
	Page     int                         `json:"page"`      // 页码
	PageSize int                         `json:"page_size"` //
}

//var SearchDataType = map[string]any{
//	"article": models.Article,
//	"chat":    models.Chat,
//	"job":     models.Job,
//}

func (s SearchController) Index(c *gin.Context) {

	var req SearchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, FailMsg(err.Error()))
		return
	}

	//c.JSON(200,DataMsg(result))
	return
}
