//go:build linux
// +build linux

package wakeevent

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"awake/pkg/logger"
)

// linuxDeviceMonitor Linux设备监听器
type linuxDeviceMonitor struct {
	handler Handler
	done    chan struct{}
}

// newPlatformDeviceMonitor 创建Linux平台的设备监听器
func newPlatformDeviceMonitor(handler Handler) DeviceMonitor {
	return &linuxDeviceMonitor{
		handler: handler,
		done:    make(chan struct{}),
	}
}

// Start 启动监听
func (m *linuxDeviceMonitor) Start() error {
	go func() {
		// 监听 /dev/input/event* 设备
		for {
			select {
			case <-m.done:
				return
			default:
				// 使用 inotify 监控 /dev/input 目录
				cmd := exec.Command("inotifywait", "-m", "-e", "access", "/dev/input/event*")
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					logger.Error("创建inotifywait管道失败: %v", err)
					return
				}

				if err := cmd.Start(); err != nil {
					logger.Error("启动inotifywait命令失败: %v", err)
					return
				}

				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					line := scanner.Text()
					if strings.Contains(line, "event") {
						// 检查设备类型
						deviceFile := strings.Fields(line)[0]
						deviceType := m.getDeviceType(deviceFile)
						if deviceType != "" {
							m.handler.HandleWakeEvent(Event{
								Type:      EventType(deviceType),
								Timestamp: time.Now(),
								Source:    deviceFile,
							})
						}
					}
				}

				cmd.Process.Kill()
				time.Sleep(time.Second) // 避免过于频繁的重启
			}
		}
	}()

	return nil
}

// getDeviceType 获取设备类型
func (m *linuxDeviceMonitor) getDeviceType(deviceFile string) string {
	// 读取设备信息
	cmd := exec.Command("udevadm", "info", "--query=property", "--name="+deviceFile)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	info := string(output)
	if strings.Contains(info, "ID_INPUT_KEYBOARD=1") {
		return string(EventTypeKeyboard)
	} else if strings.Contains(info, "ID_INPUT_MOUSE=1") {
		return string(EventTypeMouse)
	}
	return ""
}

// Stop 停止监听
func (m *linuxDeviceMonitor) Stop() error {
	close(m.done)
	return nil
}
