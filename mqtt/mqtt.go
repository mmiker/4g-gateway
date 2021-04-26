package mqtt

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/zsy-cn/4g-gateway/config"
	"github.com/zsy-cn/4g-gateway/model"
	"github.com/zsy-cn/4g-gateway/pkg/logger"
	"github.com/zsy-cn/4g-gateway/pkg/mqtt"
	"gorm.io/gorm"
)

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message, args ...interface{}) {
	// ToDo 改 mqtt 包增加 log
	// log := args[0].(*logger.Logger)
	// log.WithFields(logger.Fields{
	// 	"mqtt": "message",
	// }).Info("Received message: ", msg.Payload(), "from topic: ", msg.Topic())
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client, args ...interface{}) {
	// ToDo 改 mqtt 包增加 log
	// log := args[0].(*logger.Logger)
	// log.WithFields(logger.Fields{
	// 	"mqtt": "message",
	// }).Info("Connected")
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error, args ...interface{}) {
	// ToDo 改 mqtt 包增加 log
	// log := args[0].(*logger.Logger)
	// log.WithFields(logger.Fields{
	// 	"mqtt": "message",
	// }).Info("Connect lost:", err)
	fmt.Printf("Connect lost: %v", err)
}

func publish(log *logger.Logger, client mqtt.Client, msg string, topic string) {
	log.WithFields(logger.Fields{
		"mqtt": "publish",
	}).Info("MQTT pub message: ", msg)
	token := client.Publish(topic, 2, false, msg)
	token.Wait()
}

func NewTLSConfig(log *logger.Logger, mqttConfig *config.MQTTConfig) *tls.Config {
	cert, err := tls.LoadX509KeyPair(mqttConfig.X509Pem, mqttConfig.X509Key)
	if err != nil {
		log.WithFields(logger.Fields{
			"mqtt": "X509",
		}).Error("MQTT X509 : ", err)
	}
	return &tls.Config{
		ClientAuth:         tls.NoClientCert,
		ClientCAs:          nil,
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	}
}

// IniMQTT 初始化 MQTT
func InitMQTT(log *logger.Logger, mqttConfig *config.MQTTConfig) (client mqtt.Client, err error) {
	log.WithFields(logger.Fields{
		"mqtt": "init",
	}).Info("Init MQTT")

	log.WithFields(logger.Fields{
		"mqtt": "init",
	}).Info("mqttConfig:", mqttConfig)

	mqttClientOptions := mqtt.NewClientOptions()
	if mqttConfig.X509 {
		mqttClientOptions.AddBroker(fmt.Sprintf("tls://%s:%d", mqttConfig.Server, mqttConfig.Port))
		tlsconfig := NewTLSConfig(log, mqttConfig)
		mqttClientOptions.SetTLSConfig(tlsconfig)
	}

	if !mqttConfig.X509 {
		mqttClientOptions.AddBroker(fmt.Sprintf("tcp://%s:%d", mqttConfig.Server, mqttConfig.Port))
	}

	mqttClientOptions.SetClientID(mqttConfig.ClientID)
	mqttClientOptions.SetUsername(mqttConfig.Username)
	mqttClientOptions.SetPassword(mqttConfig.Password)
	mqttClientOptions.SetCleanSession(false)
	mqttClientOptions.SetOrderMatters(true)
	mqttClientOptions.SetProtocolVersion(4)
	mqttClientOptions.SetAutoReconnect(true)
	mqttClientOptions.SetConnectRetry(true)
	mqttClientOptions.SetStore(mqtt.NewFileStore(mqttConfig.FileStore))
	// mqttClientOptions.SetStore(NewStore())
	mqttClientOptions.SetKeepAlive(time.Duration(mqttConfig.KeepAlive) * time.Second)
	mqttClientOptions.SetDefaultPublishHandler(messagePubHandler)
	mqttClientOptions.SetOnConnectHandler(connectHandler)
	mqttClientOptions.SetConnectionLostHandler(connectLostHandler)
	log.WithFields(logger.Fields{
		"mqtt": "init",
	}).Info("配置完成")

	NewMQTTClient := mqtt.NewClient(mqttClientOptions)
	if token := NewMQTTClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	return NewMQTTClient, nil
}

// Run MQTT Publish
func Run(
	log *logger.Logger,
	db *gorm.DB,
	mqttConfig *config.MQTTConfig,
) {
	client, err := InitMQTT(log, mqttConfig)
	if err != nil {
		log.WithFields(logger.Fields{
			"system": "init",
		}).Panic("Init MQTT client: ", err)
	}
	defer client.Disconnect(250)

	log.WithFields(logger.Fields{
		"mqtt": "run",
	}).Info("MQTT RUN")

	if client == nil {
		log.WithFields(logger.Fields{
			"mqtt": "run",
		}).Error("MQTT client is null")
	}

	if mqttConfig == nil {
		log.WithFields(logger.Fields{
			"mqtt": "run",
		}).Error("MQTT config is null")
	}

	var mqttMsg model.MQTTMsg

	for {
		// 不断读取数据库，然后发送
		// 如果数据库没有数据，等待一个周期
		db.First(&mqttMsg)
		if mqttMsg.Topic == "Status" {
			db.Delete(&mqttMsg, 1)
			go publish(log, client, mqttMsg.Msg, mqttConfig.TopicBootUp)
		}
		if mqttMsg.Topic == "GPS" {
			db.Delete(&mqttMsg, 1)
			go publish(log, client, mqttMsg.Msg, mqttConfig.TopicGPS)
		}
		if mqttMsg.Topic == "" {
			time.Sleep(150 * time.Second)
		}
	}
}
