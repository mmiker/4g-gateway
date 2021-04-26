# EC20 GPS 使用方法

EC20 是移远一款带有 GPS 功能的 LTE 模块，本文主要讲述此模块 GPS 功能的基本使用方法，更多详细的指令操作细节、参数、示例可参考官网提供的手册:Quectel_EC20_GNSS_AT_Commands_Manual_V1.1

## 端口说明

EC20 挂载系统成功后，在 Windows 环境下会有三个 com 口，分别为 AT Port、DM Port、NMEA Port。其中 AT Port 用于 AT 指令的收发，而 NMEA Port 用于 GPS NMEA 数据的接收。
在 Linux 系统下，EC20 被成功识别并加载后，会有四个/dev/ttyUSBx 设备文件，ttyUSB2 用于 AT 指令收发，ttyUSB1 用于 GPS NMEA 的接收。

## GPS 功能启用步骤及说明

1、使用 AT+QGPSCFG 对 GPS 参数进行配置，此步骤不进行亦可使用（采用默认参数）。
2、使用 AT+QGPS 开启 GPS 功能，激活 NMEA 端口开始上报 GPS NMEA 数据，也可通过 AT 指令端口获取 NMEA GPS 数据。
3、若设置了 fixcount 为非 0，则当上报次数计满时会自动停止上报，若没有设置 fixcount，则可以使用 AT+QGPSEND 结束 GPS 会话。

## 主要 AT 指令及说明

指令 ｜功能｜示例｜其他说明
--- ｜---｜---｜---
AT+QGPSCFG ｜用于进行 GPS 功能的配置｜ AT+QGPSCFG="gpsnmeatype",1 (配置 nmea 格式为 GGA)｜ 具体配置请参考官网数据手册。
AT+QGPS ｜ GPS 会话配置及启动 ｜ AT+QGPS=1 (启动 GPS 会话)｜会话其他参数的配置请参考官方数据格式。
AT+QGPSEND ｜ 结束 GPS 会话 ｜ AT+QGPSEND (结束 GPS 会话 nmea 端口停止上报)｜ ——
AT+QGPSLOC ｜ 通过 AT 指令端口获取位置信息｜ AT+QGPSLOC=? (从 AT 指令端口返回位置信息)｜数据格式请参考官方数据格式。

以上只列举了简单启用 EC20 模块 GPS 功能并获取到 NMEA 数据所需的 AT 指令说明，其他功能诸如节能模式、其他定位系统模式、频次控制等操作可通过官网 GNSS AT 指令手册了解。

配置 GPS 模块时要先关闭 GPS 模块

```at
AT+QGPSEND // 关闭模块
AT+QGPS=1,,,,3 // 配置时间为3秒输出
```

## 系统应用

1、若不使用 AT+QGPSCFG 指令对 EC20 进行配置，则会以默认参数开启 GPS 参数，NMEA 端口开始上报， "gpsnmeatype"默认值为 31，上报间隔为 1s，每次上报所有种类的 NMEA 数据(GGA\RMC\GSV\GSA\VTG)，若采用此默认配置，大多数使用者会觉得单次上报的数据太多且很多信息重复，建议大家使用 QGPSCFG 配置自己需要的 NMEA 数据格式，具体格式的差异可参考网上对 NMEA 数据的说明。
2、Linux 环境下对 NMEA 数据的获取：

```sh
cat /dev/ttyUSB1 &                     // NMEA数据从ttyUSB1输出
echo -e "AT+QGPSEND\r" > /dev/ttyUSB2
echo -e "AT+QGPSCFG=\"gpsnmeatype\",1\r" > /dev/ttyUSB2
echo -e "AT+QGPS=1,,,,10\r" > /dev/ttyUSB2 // 开启GPS会话
```

可观察到 ttyUSB1 输出 NMEA 数据，如下：

```sh
$GPVTG,123.4,T,125.7,M,0.0,N,0.0,K,A*26
$GPRMC,075835.00,A,2231.527159,N,11356.035560,E,0.0,123.4,211117,2.3,W,A*21
$GPGSA,A,2,10,12,15,18,20,21,24,25,32,,,,1.0,0.7,0.8*33
$GPGSV,3,1,12,10,36,327,33,12,28,113,32,15,20,060,43,18,66,354,26*79
$GPGSV,3,2,12,20,35,119,26,21,46,215,29,24,48,035,40,25,23,156,31*70
$GPGSV,3,3,12,31,01,217,18,32,22,285,27,14,11,271,,51,,,34*4D
$GPGGA,075836.00,2231.527167,N,11356.035581,E,1,09,0.7,50.4,M,-1.0,M,,*48
```

3、程序设计过程中，若有固定频率更新位置需求，可考虑采用读取 NMEA 端口数据的形式，并将其配置适合自己需求的 NMEA 格式和数据更新间隔。若产品执行获取位置指令的频率较低且间隔时间不固定，也可考虑直接在 AT 指令端口使用 AT+QGPSLOC 指令进行实时位置信息的获取。
