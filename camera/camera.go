package camera

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zsy-cn/4g-gateway/config"
	"github.com/zsy-cn/4g-gateway/gpio"
	"github.com/zsy-cn/4g-gateway/model"
	"github.com/zsy-cn/4g-gateway/pkg/logger"
	"github.com/zsy-cn/4g-gateway/pkg/serial"
	"gorm.io/gorm"
)

// MAXRWLEN 二维码最大长度
const MAXRWLEN = 2048

// BootUp mqtt
type BootUp struct {
	Time    time.Time `json:"time"`
	Account string    `json:"account"`
	Status  int       `json:"status"`
}

// Qrcode qrcode
type Qrcode struct {
	Time    int64  `json:"time"`
	Account string `json:"account"`
}

// CameraConfig 扫码配置
type CameraConfig struct {
	DB                   *gorm.DB
	ControlGPIO          *gpio.Pin
	RunningGPIO          *gpio.Pin
	CameraSerialPort     *serial.Port
	CloseDevicePeriod    int
	QRCloseDevicePeriod  int
	QRCodeExpirationTime int
	RoleID               string
	TempRoleID           string
}

type DeviceState string                                 // 状态
type DeviceEvent string                                 // 事件
type DeviceHandler func(args []interface{}) DeviceState // 处理方法，并返回新的状态

// Device 控制流程状态机
type Device struct {
	Mu       sync.Mutex                                    // 排他锁
	State    DeviceState                                   // 当前状态
	Handlers map[DeviceState]map[DeviceEvent]DeviceHandler // 处理函数，每一个状态都可以出发有限个事件，执行有限个处理
}

// 获取当前状态
func (d *Device) getState() DeviceState {
	return d.State
}

// 设置当前状态
func (d *Device) setState(newState DeviceState) {
	d.State = newState
}

// 某状态添加事件处理方法
func (d *Device) AddHandler(state DeviceState, event DeviceEvent, handler DeviceHandler) *Device {
	if _, ok := d.Handlers[state]; !ok {
		d.Handlers[state] = make(map[DeviceEvent]DeviceHandler)
	}
	if _, ok := d.Handlers[state][event]; ok {
		fmt.Printf("[WARN] State(%s)Event(%s)已定义过", state, event)
	}
	d.Handlers[state][event] = handler
	return d
}

// 事件处理
func (d *Device) Call(event DeviceEvent, args ...interface{}) DeviceState {
	log := args[1].(*logger.Logger)
	d.Mu.Lock()
	defer d.Mu.Unlock()
	events := d.Handlers[d.getState()]
	if events == nil {
		return d.getState()
	}
	if fn, ok := events[event]; ok {
		oldState := d.getState()
		d.setState(fn(args))
		newState := d.getState()
		log.WithFields(logger.Fields{
			"camera": "call",
		}).Info("状态从 [", oldState, "] 变成 [", newState, "]")
	}
	return d.getState()
}

var (
	InterQRCode         = DeviceState("进入扫码")
	SuccessQRCode       = DeviceState("扫码成功")
	OpenDevice          = DeviceState("开机")
	CloseDevice         = DeviceState("正常关机")
	OvertimeCloseDevice = DeviceState("超时关机")
	CloseDeviceTime     = DeviceState("判断关机时间，关机时间超过要求，上传关机")

	InterQRCodeEvent         = DeviceEvent("进入扫码")
	SuccessQRCodeEvent       = DeviceEvent("设备通电")
	OpenDeviceEvent          = DeviceEvent("开机")
	CloseDeviceEvent         = DeviceEvent("正常关机")
	OvertimeCloseDeviceEvent = DeviceEvent("超时关机")

	InterQRCodeHandler = DeviceHandler(func(args []interface{}) DeviceState {
		cameraConfig := args[0].(*CameraConfig)
		log := args[1].(*logger.Logger)
		db := args[2].(*gorm.DB)
		accountold := cameraConfig.TempRoleID

		log.WithFields(logger.Fields{
			"camera": "serial",
		}).Info("开始扫码!")

		for {
			buffer := make([]byte, MAXRWLEN)
			var tmpstr string = ""
			for {
				num, err := cameraConfig.CameraSerialPort.Read(buffer)
				if err != nil {
					log.WithFields(logger.Fields{
						"camera": "serial",
					}).Error("摄像头串口读取失败: %v", err)
				}
				if num > 0 {
					tmpstr += string(buffer[:num])
				}
				// 查找读到信息的结尾标志
				if strings.LastIndex(tmpstr, "\r") > 0 {
					break
				}
			}
			// 打印输出读到的信息
			log.WithFields(logger.Fields{
				"camera": "serial",
			}).Info("摄像头读取内容: %s", tmpstr)

			decoded, err := base64.StdEncoding.DecodeString(tmpstr)
			if err != nil {
				log.WithFields(logger.Fields{
					"camera": "serial",
				}).Error("QR 格式化错误")
			}

			decodestr := string(decoded)
			log.WithFields(logger.Fields{
				"camera": "serial",
			}).Info("decodestr qrcode: %s", decodestr)

			tempQ := Qrcode{}
			err = json.Unmarshal(decoded, &tempQ)
			if err != nil {
				log.WithFields(logger.Fields{
					"camera": "serial",
				}).Error("QR 格式化错误")
			}

			if time.Now().Unix()-(tempQ.Time/1000) > int64(cameraConfig.QRCodeExpirationTime) {
				log.WithFields(logger.Fields{
					"camera": "serial",
				}).Info("二维码过期!")
				continue
			} else {
				log.WithFields(logger.Fields{
					"camera": "serial",
				}).Info("二维码正常!")
			}

			accountnew := tempQ.Account
			if accountold == accountnew {
				log.WithFields(logger.Fields{
					"camera": "serial",
				}).Info("重复扫码!")
				continue
			}

			accountold = accountnew

			if strings.Contains(decodestr, cameraConfig.RoleID) {
				Q := Qrcode{}
				err = json.Unmarshal(decoded, &Q)
				if err != nil {
					log.WithFields(logger.Fields{
						"camera": "serial",
					}).Error("MQTT 格式化错误!")
					continue
				}
				B := &BootUp{}
				B.Time = time.Now()
				B.Status = 1
				B.Account = Q.Account
				mqttData, err := json.Marshal(B)
				if err != nil {
					log.WithFields(logger.Fields{
						"camera": "serial",
					}).Error("MQTT 格式化错误!")
					continue
				}
				// 开机
				cameraConfig.ControlGPIO.Write(gpio.HIGH)
				log.WithFields(logger.Fields{
					"camera": "serial",
				}).Info("开机电源接通!")
				log.WithFields(logger.Fields{
					"status": "1",
				}).Info("成功扫码，写入 MQTT 信息!")
				db.Create(&model.MQTTMsg{Topic: "Status", Msg: string(mqttData)})
				//time.Sleep(time.Duration(5) * time.Second)
				return SuccessQRCode
			}
		}
	})

	SuccessQRCodeHandler = DeviceHandler(func(args []interface{}) DeviceState {
		cameraConfig := args[0].(*CameraConfig)
		log := args[1].(*logger.Logger)
		timeA := time.Now().Unix()

		value, err := cameraConfig.RunningGPIO.Read()
		if err != nil {
			log.WithFields(logger.Fields{
				"camera": "gpio",
			}).Error("GPIO 读取错误!")
		}

		for {
			if int(value) == 0 {
				return OpenDevice
			}
			value, err = cameraConfig.RunningGPIO.Read()
			if err != nil {
				log.WithFields(logger.Fields{
					"camera": "gpio",
				}).Error("GPIO 读取错误!")
			}
			timeB := time.Now().Unix()
			// 判断通电时间，超过通电开机时间，返回设备超时关机状态
			if (timeB - timeA) > int64(cameraConfig.QRCloseDevicePeriod) {
				return OvertimeCloseDevice
			}
		}
	})

	OpenDeviceHandler = DeviceHandler(func(args []interface{}) DeviceState {
		log := args[0].(*logger.Logger)
		db := args[1].(*gorm.DB)
		B := &BootUp{}
		B.Time = time.Now()
		B.Status = 2
		log.WithFields(logger.Fields{
			"status": "2",
		}).Info("写入 MQTT 开机!")
		mqttData, err := json.Marshal(B)
		if err != nil {
			log.WithFields(logger.Fields{
				"status": "2",
			}).Error("MQTT 格式化错误!")
		}
		db.Create(&model.MQTTMsg{Topic: "Status", Msg: string(mqttData)})
		return CloseDevice
	})

	CloseDeviceHandler = DeviceHandler(func(args []interface{}) DeviceState {
		cameraConfig := args[0].(*CameraConfig)
		log := args[1].(*logger.Logger)
		db := args[2].(*gorm.DB)
		value, err := cameraConfig.RunningGPIO.Read()
		if err != nil {
			log.WithFields(logger.Fields{
				"camera": "gpio",
			}).Error("GPIO 读取错误!")
		}
		// 上传设备关机信息，上传状态码3
		for {
			if int(value) == 0 {
				time.Sleep(time.Duration(10) * time.Second)
				value, err = cameraConfig.RunningGPIO.Read()
				if err != nil {
					log.WithFields(logger.Fields{
						"camera": "gpio",
					}).Error("GPIO 读取错误!")
				}
				continue
			}

			if int(value) == 1 {
				B := &BootUp{}
				B.Time = time.Now()
				B.Status = 3
				log.WithFields(logger.Fields{
					"status": "3",
				}).Info("写入 MQTT 关机!")
				mqttData, err := json.Marshal(B)
				if err != nil {
					log.WithFields(logger.Fields{
						"status": "3",
					}).Error("MQTT 格式化错误!")
				}
				db.Create(&model.MQTTMsg{Topic: "Status", Msg: string(mqttData)})
				return CloseDeviceTime
			}
		}
	})

	OvertimeCloseDeviceHandler = DeviceHandler(func(args []interface{}) DeviceState {
		cameraConfig := args[0].(*CameraConfig)
		log := args[1].(*logger.Logger)
		db := args[2].(*gorm.DB)
		B := &BootUp{}
		B.Time = time.Now()
		B.Status = 0
		log.WithFields(logger.Fields{
			"status": "0",
		}).Info("写入 MQTT 超时关闭设备!")

		mqttData, err := json.Marshal(B)
		if err != nil {
			log.WithFields(logger.Fields{
				"status": "0",
			}).Error("MQTT 格式化错误!")
		}
		cameraConfig.ControlGPIO.Write(gpio.LOW)
		db.Create(&model.MQTTMsg{Topic: "Status", Msg: string(mqttData)})
		return InterQRCode
	})

	CloseDeviceTimeHandler = DeviceHandler(func(args []interface{}) DeviceState {
		cameraConfig := args[0].(*CameraConfig)
		log := args[1].(*logger.Logger)
		timeC := time.Now().Unix()
		value, err := cameraConfig.RunningGPIO.Read()
		if err != nil {
			log.WithFields(logger.Fields{
				"camera": "gpio",
			}).Error("GPIO读取错误!")
		}

		for {

			if value == 0 {
				log.WithFields(logger.Fields{
					"camera": "gpio",
				}).Info("设备重新开机!")
				return OpenDevice
			}

			if value == 1 {
				if (time.Now().Unix() - timeC) >= int64(cameraConfig.CloseDevicePeriod) {
					log.WithFields(logger.Fields{
						"status": "3",
					}).Info("关闭电源!")
					cameraConfig.ControlGPIO.Write(gpio.LOW)
					return InterQRCode
				}
			}

			time.Sleep(time.Duration(10) * time.Second)

			value, err = cameraConfig.RunningGPIO.Read()
			if err != nil {
				log.WithFields(logger.Fields{
					"camera": "gpio",
				}).Error("GPIO读取错误!")
			}
		}
	})
)

// 实例化
func NewCameraDevice(initState DeviceState) *Device {
	return &Device{
		State:    initState,
		Handlers: make(map[DeviceState]map[DeviceEvent]DeviceHandler),
	}
}

// IniCamera 初始化摄像头
func InitCamera(
	log *logger.Logger,
	cameraPort string,
) (SerialPortCamera *serial.Port, err error) {
	log.WithFields(logger.Fields{
		"camera": "init",
	}).Info("Init camera")

	if len(cameraPort) == 0 {
		cameraPort = "/dev/ttyUSB0"
	}

	// 初始化串口
	serialConfig := &serial.Config{
		Name: cameraPort,
		Baud: 115200,
	}

	SerialPortCamera, err = serial.OpenPort(serialConfig)
	if err != nil {
		log.WithFields(logger.Fields{
			"camera": "serial",
		}).Fatal("serial.Open: %v", err)
	}

	configSuccess := []byte{0x02, 0x00, 0x00, 0x01, 0x00, 0x33, 0x31}
	buf := make([]byte, 2048)

	log.WithFields(logger.Fields{
		"camera": "init",
	}).Info("Config camera")

	// 配置灯颜色
	configLight1 := []byte{0x7E, 0x00, 0x08, 0x01, 0x00, 0x1B, 0x0A, 0xAB, 0xCD}
	configLight2 := []byte{0x7E, 0x00, 0x08, 0x01, 0x00, 0x1C, 0x00, 0xAB, 0xCD}
	configLight3 := []byte{0x7E, 0x00, 0x08, 0x01, 0x00, 0x1D, 0x00, 0xAB, 0xCD}
	configLight4 := []byte{0x7E, 0x00, 0x08, 0x01, 0x00, 0x1E, 0x00, 0xAB, 0xCD}
	_, err = SerialPortCamera.Write(configLight1)
	if err != nil {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("light1 Write err: ", err)
		return nil, err
	}
	time.Sleep(1 * time.Second)
	num, err := SerialPortCamera.Read(buf)
	if err != nil {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("light1 Read err: ", err)
		return nil, err
	}
	s := bytes.Equal(configSuccess, buf[:num])
	if !s {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("light1 Config err: ", err)
		return nil, err
	}
	_, err = SerialPortCamera.Write(configLight2)
	if err != nil {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("light2 Write err: ", err)
		return nil, err
	}
	time.Sleep(1 * time.Second)
	num, err = SerialPortCamera.Read(buf)
	if err != nil {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("light2 Read err: ", err)
		return nil, err
	}
	s = bytes.Equal(configSuccess, buf[:num])
	if !s {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("light2 Config err: ", err)
		return nil, err
	}
	_, err = SerialPortCamera.Write(configLight3)
	if err != nil {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("light3 Write err: ", err)
		return nil, err
	}
	time.Sleep(1 * time.Second)
	num, err = SerialPortCamera.Read(buf)
	if err != nil {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("light3 Read err: ", err)
		return nil, err
	}
	s = bytes.Equal(configSuccess, buf[:num])
	if !s {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("light3 Config err: ", err)
		return nil, err
	}
	_, err = SerialPortCamera.Write(configLight4)
	if err != nil {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("light4 Write err: ", err)
		return nil, err
	}
	time.Sleep(1 * time.Second)
	num, err = SerialPortCamera.Read(buf)
	if err != nil {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("light4 Read err: ", err)
		return nil, err
	}
	s = bytes.Equal(configSuccess, buf[:num])
	if !s {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("light4 Config err: ", err)
		return nil, err
	}
	// 配置结束符
	CR := []byte{0x7E, 0x00, 0x08, 0x01, 0x00, 0x60, 0x01, 0xAB, 0xCD}
	_, err = SerialPortCamera.Write(CR)
	if err != nil {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("CR Write err: ", err)
		return nil, err
	}
	time.Sleep(1 * time.Second)
	num, err = SerialPortCamera.Read(buf)
	if err != nil {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("CR Read err: ", err)
		return nil, err
	}
	s = bytes.Equal(configSuccess, buf[:num])
	if !s {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("CR Config err: ", err)
		return nil, err
	}
	// 保存设置
	saveConfig := []byte{0x7E, 0x00, 0x09, 0x01, 0x00, 0x00, 0x00, 0xDE, 0xC8}
	_, err = SerialPortCamera.Write(saveConfig)
	if err != nil {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("Save config Write err: ", err)
		return nil, err
	}
	time.Sleep(1 * time.Second)
	num, err = SerialPortCamera.Read(buf)
	if err != nil {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("Save config Read err: ", err)
		return nil, err
	}
	s = bytes.Equal(configSuccess, buf[:num])
	if !s {
		log.WithFields(logger.Fields{
			"camera": "config",
		}).Error("Save config err: ", err)
		return nil, err
	}

	return SerialPortCamera, nil
}

// Run 开始识别
func Run(
	log *logger.Logger,
	db *gorm.DB,
	systemConfig *config.SystemConfig,
) {
	// 初始化摄像头
	SerialPortCamera, err := InitCamera(log, systemConfig.CameraPort)
	if err != nil {
		log.WithFields(logger.Fields{
			"system": "init",
		}).Panic("Init Camera: ", err)
	}
	defer SerialPortCamera.Close()

	log.WithFields(logger.Fields{
		"camera": "run",
	}).Info("run camera")

	if systemConfig == nil || systemConfig.ControlGPIO == 0 || systemConfig.RunningGPIO == 0 {
		log.WithFields(logger.Fields{
			"camera": "run",
		}).Fatal("system config is null")
	}

	// 初始化 GPIO
	pOut, err := gpio.OpenPin(systemConfig.ControlGPIO, gpio.OUT)
	if err != nil {
		log.WithFields(logger.Fields{
			"camera": "gpio",
		}).Error("GPIO 初始化错误: %v", err)
		pOut, err = gpio.OpenPin(systemConfig.ControlGPIO, gpio.OUT)
		if err != nil {
			log.WithFields(logger.Fields{
				"camera": "gpio",
			}).Error("GPIO 重新初始化错误: %v", err)
		}
	}
	pIn, err := gpio.OpenPin(systemConfig.RunningGPIO, gpio.IN)
	if err != nil {
		log.WithFields(logger.Fields{
			"camera": "gpio",
		}).Error("GPIO 初始化错误: %v", err)
		pIn, err = gpio.OpenPin(systemConfig.RunningGPIO, gpio.IN)
		if err != nil {
			log.WithFields(logger.Fields{
				"camera": "gpio",
			}).Error("GPIO 重新初始化错误: %v", err)
		}
	}

	// 关闭电源
	pOut.Write(gpio.LOW)
	defer pOut.Close()
	defer pIn.Close()

	cameraConfig := &CameraConfig{
		DB:                   db,
		ControlGPIO:          pOut,
		RunningGPIO:          pIn,
		CameraSerialPort:     SerialPortCamera,
		CloseDevicePeriod:    systemConfig.CloseDevicePeriod,
		QRCloseDevicePeriod:  systemConfig.QRCloseDevicePeriod,
		QRCodeExpirationTime: systemConfig.QRCodeExpirationTime,
		RoleID:               systemConfig.RoleID,
		TempRoleID:           "12345678",
	}

	cameraDevice := NewCameraDevice(InterQRCode)
	cameraDevice.AddHandler(InterQRCode, InterQRCodeEvent, InterQRCodeHandler)
	cameraDevice.AddHandler(SuccessQRCode, SuccessQRCodeEvent, SuccessQRCodeHandler)
	cameraDevice.AddHandler(OpenDevice, OpenDeviceEvent, OpenDeviceHandler)
	cameraDevice.AddHandler(CloseDevice, CloseDeviceEvent, CloseDeviceHandler)
	cameraDevice.AddHandler(OvertimeCloseDevice, OvertimeCloseDeviceEvent, OvertimeCloseDeviceHandler)
	cameraDevice.AddHandler(CloseDeviceTime, CloseDeviceEvent, CloseDeviceTimeHandler)

	for {
		switch cameraDevice.State {
		case InterQRCode:
			cameraDevice.Call(InterQRCodeEvent, cameraConfig, log, db)
			continue
		case SuccessQRCode:
			cameraDevice.Call(SuccessQRCodeEvent, cameraConfig, log)
			continue
		case OpenDevice:
			cameraDevice.Call(OpenDeviceEvent, log, db)
			continue
		case CloseDevice:
			cameraDevice.Call(CloseDeviceEvent, cameraConfig, log, db)
			continue
		case OvertimeCloseDevice:
			cameraDevice.Call(OvertimeCloseDeviceEvent, cameraConfig, log, db)
			continue
		case CloseDeviceTime:
			cameraDevice.Call(CloseDeviceEvent, cameraConfig, log)
			continue
		}
	}
}
