package main

import (
	"log"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/zsy-cn/4g-gateway/camera"
	"github.com/zsy-cn/4g-gateway/config"
	"github.com/zsy-cn/4g-gateway/ec20"
	"github.com/zsy-cn/4g-gateway/geo"
	"github.com/zsy-cn/4g-gateway/model"
	"github.com/zsy-cn/4g-gateway/mqtt"
	"github.com/zsy-cn/4g-gateway/pkg/cli"
	"github.com/zsy-cn/4g-gateway/pkg/lfshook"
	"github.com/zsy-cn/4g-gateway/pkg/logger"
	"github.com/zsy-cn/4g-gateway/pkg/rotatelogs"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NetWorkStatus 判断联网状态
// func NetWorkStatus() bool {
// 	cmd := exec.Command("ping", "baidu.com", "-c", "1", "-W", "5")
// 	err := cmd.Run()
// 	if err != nil {
// 		fmt.Println(err.Error())
// 		return false
// 	}
// 	return true
// }

// options 是系统配置的结构体
type options struct {
	ConfigFile  string // 配置文件
	LogFilePath string // log 文件路径
	LogFileName string // log 文件名称
}

// Option 定义配置项
type Option func(*options)

func SetConfigFile(s string) Option {
	return func(o *options) {
		o.ConfigFile = s
	}
}

func SetLogFilePath(s string) Option {
	return func(o *options) {
		o.LogFilePath = s
	}
}

func SetLogFileName(s string) Option {
	return func(o *options) {
		o.LogFileName = s
	}
}

func InitLog(LogFilePath, LogFileName string) (log *logger.Logger, err error) {

	if len(LogFilePath) == 0 {
		LogFilePath = "./"
	}

	if len(LogFileName) == 0 {
		LogFileName = "4gGateway"
	}

	logFilePath := LogFilePath
	logFileName := LogFileName
	fileName := path.Join(logFilePath, logFileName)
	log = logger.New()
	log.SetFormatter(&logger.TextFormatter{TimestampFormat: "2006-01-02 15:04:05"})
	log.SetLevel(logger.TraceLevel)

	// 设置 rotatelogs
	logWriter, err := rotatelogs.New(
		// 分割后的文件名称
		// fileName+".%Y%m%d%H%M.log",
		fileName+".%Y%m%d.log",
		// 生成软链，指向最新日志文件
		rotatelogs.WithLinkName(fileName),
		// MaxAge and RotationCount cannot be both set  两者不能同时设置
		// 设置最大保存时间，clear 最小分钟为单位
		rotatelogs.WithMaxAge(30*24*time.Hour),
		// number 默认7份 大于7份 或到了清理时间 开始清理
		// rotatelogs.WithRotationCount(5),
		// 设置日志切割时间间隔，rotate 最小为1分钟轮询。默认60s  低于1分钟就按1分钟来
		rotatelogs.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		log.WithFields(logger.Fields{
			"log": "init",
		}).Error("logWriter rotatelogs", err)
	}

	writeMap := lfshook.WriterMap{
		logger.InfoLevel:  logWriter,
		logger.FatalLevel: logWriter,
		logger.DebugLevel: logWriter,
		logger.WarnLevel:  logWriter,
		logger.ErrorLevel: logWriter,
		logger.PanicLevel: logWriter,
	}

	lfHook := lfshook.NewHook(writeMap, &logger.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// 新增 Hook
	log.AddHook(lfHook)

	return log, nil
}

func InitDB(log *logger.Logger) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("./mqttmsg.db"), &gorm.Config{})
	if err != nil {
		log.WithFields(logger.Fields{
			"db": "init",
		}).Panic("failed to connect database: ", err)
	}
	db.AutoMigrate(&model.MQTTMsg{})
	return db
}

func RunSystem(opts []Option) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	// 设置日志
	log, err := InitLog(o.LogFilePath, o.LogFileName)
	if err != nil {
		panic(err)
	}

	// 加载环境变量
	Config, err := config.LoadINI(o.ConfigFile, log)
	if err != nil {
		log.WithFields(logger.Fields{
			"system": "init",
		}).Panic("Load config ini file: ", err)
	}

	log.Info("服务启动")
	log.Info("进程号: ", os.Getpid())

	// 初始化数据库
	db := InitDB(log)

	var wg sync.WaitGroup
	wg.Add(1)

	// 初始化4G网络
	// 判断能否联网
	go ec20.Run(log, &Config.EC20)

	// 发送 mqtt 队列
	go mqtt.Run(log, db, &Config.MQTT)

	// 打开摄像头
	// 识别qrcode编码
	// base64解码qrcode
	// 生成上电，开机，关机信息
	// 存入数据库
	// mqtt上传(时间，二维码信息)
	go camera.Run(log, db, &Config.System)

	// 获取GPS信息
	// 生成GPS上传信息
	// 存入数据库
	// mqtt上传(时间，经度，纬度)
	go geo.Run(log, db, &Config.Geo)

	wg.Wait()
}

func InitSystem(opts ...Option) error {
	state := 1
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go RunSystem(opts)

EXIT:
	for {
		sig := <-sc
		log.Printf("接收到信号[%s]\n", sig.String())

		switch sig {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			state = 0
			break EXIT
		case syscall.SIGHUP:
		default:
			break EXIT
		}
	}
	log.Println("服务退出")
	time.Sleep(time.Second)
	os.Exit(state)
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "sws gateway"
	app.Usage = "This is a SWS gateway."
	app.Commands = []*cli.Command{
		{
			Name:  "config",
			Usage: "[config file path]",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "conf",
					Aliases:  []string{"c"},
					Usage:    "配置文件(.ini)",
					Required: true,
				},
				&cli.StringFlag{
					Name:     "logPath",
					Aliases:  []string{"lp"},
					Usage:    "log 文件路径",
					Required: false,
				},
				&cli.StringFlag{
					Name:     "logName",
					Aliases:  []string{"ln"},
					Usage:    "log 文件名称",
					Required: false,
				},
			},
			Action: func(c *cli.Context) error {
				return InitSystem(
					SetConfigFile(c.String("conf")),
					SetLogFilePath(c.String("logPath")),
					SetLogFileName(c.String("logName")),
				)
			},
		},
	}
	app.Run(os.Args)
}
