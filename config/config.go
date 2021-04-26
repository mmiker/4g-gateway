package config

import (
	"fmt"

	"github.com/zsy-cn/4g-gateway/pkg/ini"
	"github.com/zsy-cn/4g-gateway/pkg/logger"
)

// Config 配置
type Config struct {
	System SystemConfig
	Log    LogConfig
	MQTT   MQTTConfig
	Geo    GeoConfig
	EC20   EC20Config
}

// SystemConfig 系统配置
type SystemConfig struct {
	ControlGPIO          int
	RunningGPIO          int
	RoleID               string
	CloseDevicePeriod    int
	QRCloseDevicePeriod  int
	QRCodeExpirationTime int
	CameraPort           string
}

// LogConfig 日志配置
type LogConfig struct {
	FileName string
	FilePath string
}

type EC20Config struct {
	DNS1   string
	DNS2   string
	Shfile string
}

// MQTTConfig MQTT 配置
type MQTTConfig struct {
	Username       string
	Password       string
	Port           int
	Server         string
	ClientID       string
	KeepAlive      int
	X509           bool
	X509Pem        string
	X509Key        string
	TopicGPS       string
	TopicHeartbeat string
	TopicVoltage   string
	TopicBootUp    string
	HeartPeriod    int
	FileStore      string
}

// GeoConfig GPS 配置
type GeoConfig struct {
	Period      int
	ControlPort string
	DataPort    string
}

// LoadINI 加载配置文件
func LoadINI(file string, log *logger.Logger) (config *Config, err error) {

	cfg, err := ini.Load(file)
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		return nil, err
	}
	defaultConfig := &Config{}
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("App Mode:", cfg.Section("").Key("app_mode").String())
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("App Version:", cfg.Section("").Key("app_version").String())

	defaultConfig.System.ControlGPIO = cfg.Section("system").Key("controlGPIO").MustInt(21)
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("System ControlGPIO:", defaultConfig.System.ControlGPIO)
	defaultConfig.System.RunningGPIO = cfg.Section("system").Key("runningGPIO").MustInt(18)
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("System RunningGPIO:", defaultConfig.System.RunningGPIO)
	defaultConfig.System.RoleID = cfg.Section("system").Key("roleID").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("System RoleID:", defaultConfig.System.RoleID)
	defaultConfig.System.CloseDevicePeriod = cfg.Section("system").Key("closeDevicePeriod").MustInt(3600)
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("System CloseDevicePeriod:", defaultConfig.System.CloseDevicePeriod)
	defaultConfig.System.QRCloseDevicePeriod = cfg.Section("system").Key("qrCloseDevicePeriod").MustInt(300)
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("System QRCloseDevicePeriod:", defaultConfig.System.QRCloseDevicePeriod)
	defaultConfig.System.QRCodeExpirationTime = cfg.Section("system").Key("qrCodeExpirationTime").MustInt(300)
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("System QRCodeExpirationTime:", defaultConfig.System.QRCodeExpirationTime)
	defaultConfig.System.CameraPort = cfg.Section("system").Key("cameraPort").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("System CameraPort:", defaultConfig.System.CameraPort)

	defaultConfig.Log.FileName = cfg.Section("log").Key("fileName").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("Log FileName:", defaultConfig.Log.FileName)
	defaultConfig.Log.FilePath = cfg.Section("log").Key("filePath").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("Log FilePath:", defaultConfig.Log.FilePath)

	defaultConfig.EC20.DNS1 = cfg.Section("ec20").Key("dns1").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("EC20 DNS1:", defaultConfig.EC20.DNS1)
	defaultConfig.EC20.DNS2 = cfg.Section("ec20").Key("dns2").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("EC20 DNS2:", defaultConfig.EC20.DNS2)
	defaultConfig.EC20.Shfile = cfg.Section("ec20").Key("shfile").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("EC20 Shfile:", defaultConfig.EC20.Shfile)

	defaultConfig.MQTT.Username = cfg.Section("mqtt").Key("username").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("MQTT Username:", defaultConfig.MQTT.Username)
	defaultConfig.MQTT.Password = cfg.Section("mqtt").Key("password").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("MQTT Password:", defaultConfig.MQTT.Password)
	defaultConfig.MQTT.Port = cfg.Section("mqtt").Key("port").MustInt(1883)
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("MQTT Port:", defaultConfig.MQTT.Port)
	defaultConfig.MQTT.X509 = cfg.Section("mqtt").Key("x509").MustBool(false)
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("MQTT X509:", defaultConfig.MQTT.X509)
	defaultConfig.MQTT.X509Pem = cfg.Section("mqtt").Key("x509Pem").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("MQTT X509Pem:", defaultConfig.MQTT.X509Pem)
	defaultConfig.MQTT.X509Key = cfg.Section("mqtt").Key("x509Key").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("MQTT X509Key:", defaultConfig.MQTT.X509Key)
	defaultConfig.MQTT.Server = cfg.Section("mqtt").Key("server").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("MQTT Server:", defaultConfig.MQTT.Server)
	defaultConfig.MQTT.ClientID = cfg.Section("mqtt").Key("clientID").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("MQTT Server:", defaultConfig.MQTT.ClientID)
	defaultConfig.MQTT.KeepAlive = cfg.Section("mqtt").Key("keepAlive").MustInt(60)
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("MQTT Port:", defaultConfig.MQTT.KeepAlive)
	defaultConfig.MQTT.TopicGPS = cfg.Section("mqtt").Key("topicGPS").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("MQTT Topic GPS:", defaultConfig.MQTT.TopicGPS)
	defaultConfig.MQTT.TopicBootUp = cfg.Section("mqtt").Key("topicBootUp").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("MQTT Topic Boot Up:", defaultConfig.MQTT.TopicBootUp)
	defaultConfig.MQTT.FileStore = cfg.Section("mqtt").Key("fileStore").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("MQTT File Store:", defaultConfig.MQTT.FileStore)

	defaultConfig.Geo.Period = cfg.Section("geo").Key("period").MustInt(100)
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("Geo Period:", defaultConfig.Geo.Period)
	defaultConfig.Geo.ControlPort = cfg.Section("geo").Key("controlPort").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("Geo Control Port:", defaultConfig.Geo.ControlPort)
	defaultConfig.Geo.DataPort = cfg.Section("geo").Key("dataPort").String()
	log.WithFields(logger.Fields{
		"config": "load",
	}).Info("Geo Data Port:", defaultConfig.Geo.DataPort)

	return defaultConfig, nil
}

// MQTTPubType MQTT 发送类型
type MQTTPubType int

// GPSMQTT 上传 GPS 信息
// StatusMQTT 上传 通电，开机，关机信息
const (
	GPSMQTT MQTTPubType = iota
	StatusMQTT
)
