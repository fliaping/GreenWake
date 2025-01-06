//go:build darwin
// +build darwin

package wakeevent

import (
	"bufio"
	"greenwake-guard/config"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"greenwake-guard/pkg/logger"
)

const (
	// ActivityCheckInterval 设备活动检查间隔
	ActivityCheckInterval = 15 * time.Second
)

// darwinDeviceMonitor macOS设备监听器
type darwinDeviceMonitor struct {
	handler Handler
	config  *config.Config
	done    chan struct{}
	// 添加监控状态标志
	monitoring atomic.Bool
	mu         sync.Mutex
}

// newPlatformDeviceMonitor 创建macOS平台的设备监听器
func newPlatformDeviceMonitor(handler Handler) DeviceMonitor {
	return &darwinDeviceMonitor{
		handler: handler,
		done:    make(chan struct{}),
	}
}

// Start 启动监听
func (m *darwinDeviceMonitor) Start() error {
	logger.Info("启动 macOS 设备监听器")
	m.monitoring.Store(true)
	go m.monitorUserActivity()
	return nil
}

// monitorUserActivity 监听用户活动
func (m *darwinDeviceMonitor) monitorUserActivity() {
	logger.Debug("开始监控用户活动, 每 %v 检查一次", ActivityCheckInterval)
	ticker := time.NewTicker(ActivityCheckInterval)
	defer ticker.Stop()

	for m.monitoring.Load() {
		select {
		case <-m.done:
			logger.Info("停止监控用户活动")
			return
		case <-ticker.C:
			logger.Debug("执行活动检查...")
			// 使用 pmset -g assertions 命令检查当前系统断言
			cmd := exec.Command("pmset", "-g", "assertions")
			output, err := cmd.Output()
			if err != nil {
				logger.Error("检查用户活动失败: %v", err)
				continue
			}
			logger.Debug("pmset 命令输出长度: %d 字节", len(output))

			// 检查是否有活动
			if hasUserActivity(string(output)) {
				now := time.Now()
				logger.Debug("检测到用户活动")
				m.handler.HandleWakeEvent(Event{
					Type:      EventTypeDevice,
					Timestamp: now,
					Source:    "user_activity",
				})
			} else {
				logger.Debug("未检测到用户活动")
			}
		}
	}
}

// hasUserActivity 检查是否有用户活动
func hasUserActivity(output string) bool {
	var hasUserIsActive bool
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "UserIsActive") {
			hasUserIsActive = true
			logger.Debug("发现 UserIsActive 标记: %s", line)
		}
	}
	return hasUserIsActive
}

// Stop 停止监听
func (m *darwinDeviceMonitor) Stop() error {
	close(m.done)
	return nil
}

// UpdateConfig 更新配置
func (m *darwinDeviceMonitor) UpdateConfig(config *config.Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = config
	if config.IsEventTypeValid(string(EventTypeDevice)) {
		// 如果当前没有监控且配置允许监控，则启动监控
		if !m.monitoring.Load() {
			m.monitoring.Store(true)
			go m.monitorUserActivity()
		}
	} else {
		// 如果正在监控但配置不允许监控，则停止监控
		if m.monitoring.Load() {
			m.monitoring.Store(false)
			close(m.done)
			m.done = make(chan struct{}) // 重新创建channel以供后续使用
		}
	}
}
