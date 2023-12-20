package database

import (
	"context"
	"fmt"
	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"greet_gin/config"
	"log"
	"os"
	"time"
)

var elasticClient *elastic.Client

// GetElasticClient 获取 elastic 连接
func GetElasticClient(ctx context.Context) *elastic.Client {
	return elasticClient
}

type ElasticSearch struct {
	MustQuery    []elastic.Query
	MustNotQuery []elastic.Query
	ShouldQuery  []elastic.Query
	Filters      []elastic.Query
	Sorters      []elastic.Sorter
	From         int //分页
	Size         int
}

func InitES() *elastic.Client {
	url := fmt.Sprintf(config.ESSetting.Url)
	elasticClient, err := elastic.NewClient(
		//elastic 服务地址
		elastic.SetURL(url),
		elastic.SetSniff(false),
		// 设置错误日志输出
		elastic.SetErrorLog(log.New(os.Stderr, "ELASTIC ", log.LstdFlags)),
		elastic.SetHealthcheckInterval(30*time.Second), //设置两次健康检查之间的间隔。默认间隔为60秒。
		// 设置info日志输出
		elastic.SetInfoLog(log.New(os.Stdout, "", log.LstdFlags)))
	if err != nil {
		logrus.Errorf("Failed to create elastic client:%v", err)
		return elasticClient
	}
	return elasticClient
}
