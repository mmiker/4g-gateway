package ec20

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/zsy-cn/4g-gateway/config"
	"github.com/zsy-cn/4g-gateway/pkg/logger"
)

type NetworkState string                                  //状态
type NetworkEvent string                                  //事件
type NetworkHandler func(args []interface{}) NetworkState //处理

//有限状态机
type Network struct {
	Mu       sync.Mutex                                       // 排他锁
	State    NetworkState                                     // 当前状态
	Handlers map[NetworkState]map[NetworkEvent]NetworkHandler // 处理函数，每一个状态都可以出发有限个事件，执行有限个处理
}

// 获取当前状态
func (n *Network) getState() NetworkState {
	return n.State
}

// 设置当前状态
func (n *Network) setState(newState NetworkState) {
	n.State = newState
}

// 某状态添加事件处理方法
func (n *Network) AddHandler(state NetworkState, event NetworkEvent, handler NetworkHandler) *Network {
	if _, ok := n.Handlers[state]; !ok {
		n.Handlers[state] = make(map[NetworkEvent]NetworkHandler)
	}
	if _, ok := n.Handlers[state][event]; ok {
		fmt.Printf("[WARN] State(%s)Event(%s)已定义过", state, event)
	}
	n.Handlers[state][event] = handler
	return n
}

// 事件处理
func (n *Network) Call(event NetworkEvent, args ...interface{}) NetworkState {
	log := args[0].(*logger.Logger)
	n.Mu.Lock()
	defer n.Mu.Unlock()
	events := n.Handlers[n.getState()]
	if events == nil {
		return n.getState()
	}
	if fn, ok := events[event]; ok {
		oldState := n.getState()
		n.setState(fn(args))
		newState := n.getState()
		log.WithFields(logger.Fields{
			"Network": "call",
		}).Info("状态从 [", oldState, "] 变成 [", newState, "]")
	}
	return n.getState()
}

func NewNetwork(initState NetworkState) *Network {
	return &Network{
		State:    initState,
		Handlers: make(map[NetworkState]map[NetworkEvent]NetworkHandler),
	}
}

var (
	Judge        = NetworkState("判断联网")
	Networked    = NetworkState("已联网")
	NotNetworked = NetworkState("未联网")
	ExistProcess = NetworkState("pppd进程存在")
	NoProcess    = NetworkState("无pppd进程")

	JudgeEvent             = NetworkEvent("判断是否联网")
	NetworkedEvent         = NetworkEvent("联网成功")
	NetworkSIMEvent        = NetworkEvent("拨号联网")
	CheckProcessEvent      = NetworkEvent("检查pppd进程")
	WaitOrKillProcessEvent = NetworkEvent("判断等待联网或杀死进程")

	JudgeHandler = NetworkHandler(func(args []interface{}) NetworkState {
		log := args[0].(*logger.Logger)
		config := args[1].(*config.EC20Config)
		link, err := IsCanPingOpen(config.DNS1)
		if err != nil {
			log.WithFields(logger.Fields{
				"network": "ping",
			}).Error("ping was wrong:", err)
		}
		if !link {
			log.WithFields(logger.Fields{
				"network": "ping",
			}).Info("network is disrupted")
			return NotNetworked
		}
		log.WithFields(logger.Fields{
			"network": "ping",
		}).Info("network is normal")
		return Networked
	})

	NetworkedHandler = NetworkHandler(func(args []interface{}) NetworkState {
		log := args[0].(*logger.Logger)
		log.WithFields(logger.Fields{
			"network": "connection",
		}).Info("network is normal")
		time.Sleep(time.Duration(90) * time.Second)
		return Judge
	})

	NetworkSIMHandler = NetworkHandler(func(args []interface{}) NetworkState {
		log := args[0].(*logger.Logger)
		config := args[1].(*config.EC20Config)
		//先授予.sh文件执行权限，再执行.sh文件
		cmdd := exec.Command("/bin/bash", "-c", "chmod u+x "+config.Shfile)
		stdoutbyte, err := cmdd.Output()
		if err != nil {
			log.WithFields(logger.Fields{
				"execution": "chmod",
			}).Error("chmod output failed:", err)
		}
		if err == nil {
			log.WithFields(logger.Fields{
				"execution": "chmod",
			}).Info("chmod succeeded:" + string(stdoutbyte))
		}
		_, err = ExecCommand("sudo ./"+config.Shfile, log)
		if err != nil {
			log.WithFields(logger.Fields{
				"execution": "sh",
			}).Error("file execution failed:", err)
		}
		time.Sleep(90 * time.Second)
		return Judge
	})

	CheckProcessHandler = NetworkHandler(func(args []interface{}) NetworkState {
		log := args[0].(*logger.Logger)
		excmd := exec.Command("/bin/bash", "-c", "ps -aux | grep pppd | grep -v grep")
		std_out, err := excmd.Output()
		if err != nil {
			log.WithFields(logger.Fields{
				"execution": "ps",
			}).Error("file execution failed:", err)
		}
		processString := string(std_out)
		if strings.Contains(processString, "pppd") {
			log.WithFields(logger.Fields{
				"process": "pppd",
			}).Info("pppd process existed")
			return ExistProcess
		}
		log.WithFields(logger.Fields{
			"process": "pppd",
		}).Info("No pppd process")
		return NoProcess
	})

	WaitOrKillProcessHandler = NetworkHandler(func(args []interface{}) NetworkState {
		log := args[0].(*logger.Logger)
		config := args[1].(*config.EC20Config)
		timeout := time.Duration(120 * time.Second)
		_, err := net.DialTimeout("tcp", config.DNS1, timeout)
		if err != nil {
			log.WithFields(logger.Fields{
				"wait": "ping",
			}).Error("ping was wrong:", err)
			excmd := exec.Command("/bin/bash", "-c", "ps -aux | grep pppd | grep -v grep")
			std_out, err := excmd.Output()
			if err != nil {
				log.WithFields(logger.Fields{
					"execution": "ps",
				}).Error("ps execution failed in Killprocess:", err)
			}
			splitString := strings.Fields(string(std_out))
			for _, fragment := range splitString {
				if strings.Contains(fragment, "pppd") {
					killCommand := "sudo kill -9 " + splitString[1]
					cmd := exec.Command("/bin/bash", "-c", killCommand)
					_, err := cmd.Output()
					if err != nil {
						log.WithFields(logger.Fields{
							"execution": "kill",
						}).Error("kill execution failed:", err)
					}
					// out_string :=string(out_bytes)
					if err == nil {
						log.WithFields(logger.Fields{
							"network": "kill",
						}).Info("kill pppd process succeeded")
						break
					}
				}
			}
			return NoProcess
		}
		log.WithFields(logger.Fields{
			"wait": "ping",
		}).Info("wait ping succeeded")
		return Networked
	})
)

//IsCanPingOpen 函数是确定是否联网的方法
func IsCanPingOpen(dns string) (bool, error) {
	timeout := time.Duration(10 * time.Second)
	_, err := net.DialTimeout("tcp", dns, timeout)
	if err != nil {
		return false, err
	}
	return true, nil
}

//ExecCommand 函数是执行.sh脚本的方法
func ExecCommand(strcommand string, log *logger.Logger) (bool, error) {
	cmd := exec.Command("/bin/sh", "-c", strcommand)
	// stdout,_ :=cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		log.WithFields(logger.Fields{
			"execution": "sh",
		}).Error("execution sh Start failed:", err)
		return false, err
	}
	// out_bytes,_ :=ioutil.ReadAll(stdout)
	// stdout.Close()
	if err := cmd.Wait(); err != nil {
		log.WithFields(logger.Fields{
			"execution": "sh",
		}).Error("execution sh Wait failed:", err)
		return false, err
	} else {
		log.WithFields(logger.Fields{
			"execution": "sh",
		}).Info("sh execution sucessful")
	}
	return true, nil
}

func Run(
	log *logger.Logger,
	ec20Config *config.EC20Config,
) {
	network := NewNetwork(Judge)
	network.AddHandler(Judge, JudgeEvent, JudgeHandler)
	network.AddHandler(Networked, NetworkedEvent, NetworkedHandler)
	network.AddHandler(NotNetworked, CheckProcessEvent, CheckProcessHandler)
	network.AddHandler(ExistProcess, WaitOrKillProcessEvent, WaitOrKillProcessHandler)
	network.AddHandler(NoProcess, NetworkSIMEvent, NetworkSIMHandler)
	for {
		switch network.State {
		case Judge:
			network.Call(JudgeEvent, log, ec20Config)
			continue
		case Networked:
			network.Call(NetworkedEvent, log)
			continue
		case NotNetworked:
			network.Call(CheckProcessEvent, log)
			continue
		case NoProcess:
			network.Call(NetworkSIMEvent, log, ec20Config)
			continue
		case ExistProcess:
			network.Call(WaitOrKillProcessEvent, log, ec20Config)
		}
	}
}
