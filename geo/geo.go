package geo

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/zsy-cn/4g-gateway/config"
	"github.com/zsy-cn/4g-gateway/model"
	"github.com/zsy-cn/4g-gateway/pkg/logger"
	"github.com/zsy-cn/4g-gateway/pkg/serial"
	"gorm.io/gorm"
)

// NMEA 符合 NMEA 规定
type NMEA struct {
	fixTimestamp       string
	latitude           string
	latitudeDirection  string
	longitude          string
	longitudeDirection string
	fixQuality         string
	satellites         string
}

// ParseNMEALine ParseNMEALine
func ParseNMEALine(line string) (NMEA, error) {
	tokens := strings.Split(line, ",")
	if tokens[0] == "$GPGGA" {
		return NMEA{
			fixTimestamp:       tokens[1],
			latitude:           tokens[2],
			latitudeDirection:  tokens[3],
			longitude:          tokens[4],
			longitudeDirection: tokens[5],
			fixQuality:         tokens[6],
			satellites:         tokens[7],
		}, nil
	}
	return NMEA{}, errors.New("unsupported nmea string")
}

// ParseDegrees ParseDegrees
func ParseDegrees(value string, direction string) (string, error) {
	if value == "" || direction == "" {
		return "", errors.New("the location and / or direction value does not exist")
	}
	lat, _ := strconv.ParseFloat(value, 64)
	degrees := math.Floor(lat / 100)
	minutes := ((lat / 100) - math.Floor(lat/100)) * 100 / 60
	decimal := degrees + minutes
	if direction == "W" || direction == "S" {
		decimal *= -1
	}
	return fmt.Sprintf("%.6f", decimal), nil
}

// GetLatitude GetLatitude
func (nmea NMEA) GetLatitude() (string, error) {
	return ParseDegrees(nmea.latitude, nmea.latitudeDirection)
}

// GetLongitude GetLongitude
func (nmea NMEA) GetLongitude() (string, error) {
	return ParseDegrees(nmea.longitude, nmea.longitudeDirection)
}

// Geo mqtt
type Geo struct {
	Time      time.Time `json:"time"`
	Latitude  string    `json:"latitude"`
	Longitude string    `json:"longitude"`
}

// IniGeo 初始化 GPS 模块
func InitGeo(
	log *logger.Logger,
	geoConfig *config.GeoConfig,
) (serialPort *serial.Port, err error) {
	log.WithFields(logger.Fields{
		"geo": "init",
	}).Info("Init geo")

	optionUSB2 := &serial.Config{
		Name:        geoConfig.ControlPort,
		Baud:        115200,
		ReadTimeout: time.Second,
	}
	serialPortUSB2, err := serial.OpenPort(optionUSB2)
	if err != nil {
		log.WithFields(logger.Fields{
			"geo": "init",
		}).Error("serial open err: ", err)
		return nil, err
	}

	QGPSCFGATCommand := "AT+QGPSCFG=\"gpsnmeatype\",1\r"
	QGPSENDATCommand := "AT+QGPSEND\r"
	QGPSATCommand := "AT+QGPS=1,,,,10\r"

	buf := make([]byte, 128)

	//b := []byte{0x41, 0x54, 0x2B, 0x51, 0x47, 0x50, 0x53, 0x3D, 0x31, 0x0d}
	_, err = serialPortUSB2.Write([]byte(QGPSENDATCommand))
	if err != nil {
		log.WithFields(logger.Fields{
			"geo": "init",
		}).Error("QGPSENDATCommand write err: ", err)
		return nil, err
	}
	time.Sleep(1 * time.Second)
	num, err := serialPortUSB2.Read(buf)
	if err != nil {
		log.WithFields(logger.Fields{
			"geo": "config",
		}).Error("QGPSENDATCommand Read err: ", err)
		return nil, err
	}
	log.WithFields(logger.Fields{
		"geo": "config",
	}).Info("QGPSENDATCommand Config: ", string(buf[:num]))

	_, err = serialPortUSB2.Write([]byte(QGPSCFGATCommand))
	if err != nil {
		log.WithFields(logger.Fields{
			"geo": "init",
		}).Error("QGPSCFGATCommand write err: ", err)
		return nil, err
	}
	time.Sleep(1 * time.Second)
	num, err = serialPortUSB2.Read(buf)
	if err != nil {
		log.WithFields(logger.Fields{
			"geo": "config",
		}).Error("QGPSCFGATCommand Read err: ", err)
		return nil, err
	}
	log.WithFields(logger.Fields{
		"geo": "config",
	}).Info("QGPSCFGATCommand Config: ", string(buf[:num]))
	time.Sleep(1 * time.Second)

	_, err = serialPortUSB2.Write([]byte(QGPSATCommand))
	if err != nil {
		log.WithFields(logger.Fields{
			"geo": "init",
		}).Error("QGPSATCommand write err: ", err)
		return nil, err
	}
	time.Sleep(1 * time.Second)
	num, err = serialPortUSB2.Read(buf)
	if err != nil {
		log.WithFields(logger.Fields{
			"geo": "config",
		}).Error("QGPSATCommand Read err: ", err)
		return nil, err
	}
	log.WithFields(logger.Fields{
		"geo": "config",
	}).Info("QGPSATCommand Config: ", string(buf[:num]))
	time.Sleep(1 * time.Second)

	serialPortUSB2.Close()

	options := &serial.Config{
		Name:        geoConfig.DataPort,
		Baud:        115200,
		ReadTimeout: time.Second,
	}
	serialPortUSB1, err := serial.OpenPort(options)
	if err != nil {
		log.WithFields(logger.Fields{
			"geo": "init",
		}).Error("serial open err: ", err)
		return nil, err
	}

	return serialPortUSB1, nil
}

// Run 采集并上传 GPS 数据
func Run(
	log *logger.Logger,
	db *gorm.DB,
	geoConfig *config.GeoConfig,
) {
	// 初始化GPS
	serialPortGPS, err := InitGeo(log, geoConfig)
	if err != nil {
		log.WithFields(logger.Fields{
			"system": "init",
		}).Panic("Init GPS: ", err)
	}
	defer serialPortGPS.Close()

	if geoConfig == nil {
		log.WithFields(logger.Fields{
			"geo": "run",
		}).Fatal("Init geo config is null")
	}

	reader := bufio.NewReader(serialPortGPS)
	scanner := bufio.NewScanner(reader)
	for {
		G := &Geo{}
		for scanner.Scan() {
			gps, err := ParseNMEALine(scanner.Text())
			log.WithFields(logger.Fields{
				"geo": "run",
			}).Info(scanner.Text())
			if err != nil {
				// log.WithFields(logger.Fields{
				// 	"geo": "run",
				// }).Error("Parse NMEA Line Err:", err)
				continue
			}
			if err == nil {
				if gps.fixQuality == "1" || gps.fixQuality == "2" {
					latitude, _ := gps.GetLatitude()
					longitude, _ := gps.GetLongitude()
					// log.WithFields(logger.Fields{
					// 	"geo": "run",
					// }).Info(latitude + "," + longitude)
					G.Time = time.Now()
					G.Latitude = latitude
					G.Longitude = longitude
					mqttData, err := json.Marshal(G)
					if err != nil {
						log.WithFields(logger.Fields{
							"geo": "run",
						}).Error("MQTT Json Marshal Err:", err)
						continue
					}
					// 存入数据库
					db.Create(&model.MQTTMsg{Topic: "GPS", Msg: string(mqttData)})
					time.Sleep(time.Duration(geoConfig.Period) * time.Second)
				} else {
					log.WithFields(logger.Fields{
						"geo": "run",
					}).Info("no gps fix available")
					time.Sleep(10 * time.Second)
				}
			}
		}
	}
}
