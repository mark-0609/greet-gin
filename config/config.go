package config

import (
	"fmt"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/pkg/errors"
	"github.com/rifflock/lfshook"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-ini/ini"
	"github.com/sirupsen/logrus"
)

var cfg *ini.File

// DatabaseSetting 数据库配置
var DatabaseSetting = &Mysql{}

// RedisSetting redis配置
var RedisSetting = &Redis{}

// LogSetting 日志配置
var LogSetting = &Log{}

// ServerSetting 服务器配置
var ServerSetting = &Server{}

var ESSetting = &ES{}

// redis配置
type Redis struct {
	Addr     string
	Password string
	DB       int
	Port     string
}

// 数据库配置
type Mysql struct {
	DbHost     string
	DbUser     string
	DbPassword string
	DbPort     string
	DbDatabase string
}

// Log 日志配置类
type Log struct {
	LogPath      string
	LogFileName  string
	MaxHour      string
	RotationHour string
	SentryDsn    string
}

// Server 服务器配置类
type Server struct {
	Debug       bool
	Environment string //dev,test,production
	Port        string
	Domain      string
	FilePath    string
}

type ES struct {
	UserName string
	Password string
	Url      string
}

const (
	// Default log format will output [INFO]: 2006-01-02T15:04:05Z07:00 - Log message
	defaultLogFormat       = "%time%|[%lvl%]|%sn%|%user%|%ip%:%port%|%func%|%msg%"
	defaultTimestampFormat = "2006-01-02 15:04:05.000"

	DefaultTimeLayOut   = "2006-01-02 15:04:05"
	ReqContextKeyUserId = "userId"
	ReqContextKeySn     = "sn"
)

func mapTo(section string, v interface{}) {
	err := cfg.Section(section).MapTo(v)
	if err != nil {
		logrus.Fatalf("Cfg.MapTo setting err: %v", err)
	}
}

// Setup 初始化配置
func Setup() {
	var err error
	cfg, err = ini.Load("config/app.ini.exmaple")
	if err != nil {
		fmt.Println("loading app.ini.exmaple fail try to run as shell script…")
		cfg, err = ini.Load("../../config/app.ini.exmaple")
		if err != nil {
			panic("loading app.ini.exmaple fail again,check path of app.ini.exmaple")
		}
		fmt.Println("loading app.ini.exmaple success")
	}

	mapTo("server", ServerSetting)
	mapTo("database", DatabaseSetting)
	mapTo("redis", RedisSetting)
	mapTo("log", LogSetting)
	mapTo("es", ESSetting)

	setupPath(LogSetting.LogPath)
	setupPath(ServerSetting.FilePath)

	setupLogrus()

	if ServerSetting.Environment != "dev" {
		gin.SetMode(gin.ReleaseMode)
	}
}

func isDirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	} else {
		return fi.IsDir()
	}
}

// 初始化目录
func setupPath(path string) {
	if isDirExists(path) {
		return
	}
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		logrus.Errorf("init file system error", errors.WithStack(err))
	}
}

// setupLogrus 初始化日志配置
func setupLogrus() {
	logrus.SetReportCaller(true)

	maxAge, err := time.ParseDuration(LogSetting.MaxHour + "h")
	if err != nil {
		fmt.Println("init log maxAge fail")
		panic(err)
	}
	rotationTime, err := time.ParseDuration(LogSetting.RotationHour + "h")
	if err != nil {
		fmt.Println("init log rotationTime fail")
		panic(err)
	}
	configLocalFilesystemLogger(LogSetting.LogPath, LogSetting.LogFileName, maxAge, rotationTime)
}

// ToggleLogFormatter 日志格式
type ToggleLogFormatter struct {
	TimestampFormat string
	// Available standard keys: time, msg, lvl, file, func
	// Also can include custom fields but limited to strings.
	// All of fields need to be wrapped inside %% i.e %time% %msg%
	LogFormat string
}

// Format 格式化返回
func (formatter *ToggleLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	output := formatter.LogFormat
	if output == "" {
		output = defaultLogFormat
	}

	timestampFormat := formatter.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}

	var funcVal, fileVal string

	if entry.HasCaller() {
		funcVal = entry.Caller.Function
		fileVal = fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
	}

	output = strings.Replace(output, "%time%", entry.Time.Format(timestampFormat), 1)
	output = strings.Replace(output, "%lvl%", strings.ToUpper(entry.Level.String()), 1)

	user := ""
	sn := ""
	ip := ""
	port := ""

	if entry.Context != nil {
		user = entry.Context.Value(ReqContextKeyUserId).(string)
		sn = entry.Context.Value(ReqContextKeySn).(string)

		//ip  = entry.Context.Value(REQ_CONTEXT_KEY_SN).(string)
		//port = entry.Context.Value(REQ_CONTEXT_KEY_SN).(string)
	}

	output = strings.Replace(output, "%sn%", sn, 1)
	output = strings.Replace(output, "%user%", user, 1)
	output = strings.Replace(output, "%ip%", ip, 1)
	output = strings.Replace(output, "%port%", port, 1)
	output = strings.Replace(output, "%file%", fileVal, 1)
	output = strings.Replace(output, "%func%", funcVal, 1)
	output = strings.Replace(output, "%msg%", entry.Message, 1)

	for k, v := range entry.Data {
		if s, ok := v.(string); ok {
			output = strings.Replace(output, "%"+k+"%", s, 1)
		}
	}
	output = output + "\n"

	return []byte(output), nil

}

// configLocalFilesystemLogger
func configLocalFilesystemLogger(logPath string, logFileName string, maxAge time.Duration, rotationTime time.Duration) {
	baseLogPath := path.Join(logPath, logFileName)
	writer, err := rotatelogs.New(
		baseLogPath+"%Y%m%d%H%M"+"_"+ServerSetting.Environment+".log",
		rotatelogs.WithLinkName(baseLogPath),      // 生成软链，指向最新日志文件
		rotatelogs.WithMaxAge(maxAge),             // 文件最大保存时间
		rotatelogs.WithRotationTime(rotationTime), // 日志切割时间间隔
	)
	mw := io.MultiWriter(os.Stdout, writer)
	gin.DefaultWriter = mw

	if err != nil {
		logrus.Errorf("config local file system logger error. %+v", errors.WithStack(err))
	}
	lfHook := lfshook.NewHook(lfshook.WriterMap{
		logrus.DebugLevel: writer, // 为不同级别设置不同的输出目的
		logrus.InfoLevel:  writer,
		logrus.WarnLevel:  writer,
		logrus.ErrorLevel: writer,
		logrus.FatalLevel: writer,
		logrus.PanicLevel: writer,
	}, &ToggleLogFormatter{})

	logrus.SetFormatter(&ToggleLogFormatter{})
	logrus.AddHook(lfHook)

	if ServerSetting.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

}
