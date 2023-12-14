package database

import (
	"fmt"
	"greet_gin/config"
	"greet_gin/models"

	"github.com/go-redis/redis"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/o1egl/gormrus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/gormigrate.v1"
)

var db *gorm.DB
var redisClient *redis.Client

// 获取redis连接
func GetRedisClient() *redis.Client {
	return redisClient
}

// 获取主数据库的数据库连接
func GetDb() *gorm.DB {
	return db
}

// db初始化
func Init() {
	fmt.Println("try connect db")
	var err error
	db, err = gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true&loc=Local",
		config.DatabaseSetting.DbUser,
		config.DatabaseSetting.DbPassword,
		config.DatabaseSetting.DbHost,
		config.DatabaseSetting.DbPort,
		config.DatabaseSetting.DbDatabase))
	if err != nil {
		log.Fatalf("cannot connect db: %v", err)
	}
	db.SingularTable(true)
	db.DB().SetMaxIdleConns(10)
	db.DB().SetMaxOpenConns(100)
	db.SetLogger(gormrus.NewWithNameAndLogger("db", log.StandardLogger()))
	db.LogMode(true)

	autoMigrate()

	redisInit()
	fmt.Println("redis success")

}

// 关闭
func CloseDB() {
	defer db.Close()
	defer redisClient.Close()
}

// redis
func redisInit() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     config.RedisSetting.Addr,
		Password: config.RedisSetting.Password,
		DB:       config.RedisSetting.DB,
	})
}

func autoMigrate() {
	log.Printf("start auto migrate")
	var err error
	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "201608301400",
			Migrate: func(tx *gorm.DB) error {
				tx.AutoMigrate(&models.User{})
				return nil
			},
			Rollback: func(tx *gorm.DB) error {
				tx.DropTable(&models.User{})
				return nil
			},
		},
	})

	if err = m.Migrate(); err != nil {
		log.Fatalf("Could not migrate: %v", err)
	}
	log.Printf("Migration did run successfully")
}
