//go:build darwin
// +build darwin

package wakeevent

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"awake/pkg/logger"
)

// darwinDeviceMonitor macOS设备监听器
type darwinDeviceMonitor struct {
	handler Handler
	done    chan struct{}
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
	go m.monitorUserActivity()
	return nil
}

// monitorUserActivity 监听用户活动
func (m *darwinDeviceMonitor) monitorUserActivity() {
	logger.Debug("开始监控用户活动, 30秒检查一次")
	ticker := time.NewTicker(30 * time.Second) // 降低检查频率
	defer ticker.Stop()

	var lastActivity time.Time
	var lastLogTime time.Time
	for {
		select {
		case <-m.done:
			logger.Info("停止监控用户活动")
			return
		case <-ticker.C:
			logger.Debug("执行活动检查...")
			// 使用 pmset 命令检查用户活动日志，并通过管道传递给 tail 命令限制输出行数
			cmd := exec.Command("sh", "-c", "pmset -g log | tail -n 1000")
			output, err := cmd.Output()
			if err != nil {
				logger.Error("检查用户活动失败: %v", err)
				continue
			}
			logger.Debug("pmset 命令输出长度: %d 字节", len(output))

			// 解析输出寻找用户活动
			if hasRecentActivity(string(output), lastLogTime) {
				now := time.Now()
				// 至少间隔5秒才触发一次事件
				if now.Sub(lastActivity) >= 5*time.Second {
					logger.Info("检测到用户活动，距离上次活动: %v", now.Sub(lastActivity))
					m.handler.HandleWakeEvent(Event{
						Type:      EventTypeKeyboard,
						Timestamp: now,
						Source:    "user_activity",
					})
					lastActivity = now
				} else {
					logger.Debug("活动间隔太短，跳过: %v", now.Sub(lastActivity))
				}
			} else {
				logger.Debug("未检测到最近的用户活动")
			}
			lastLogTime = time.Now()
		}
	}
}

// hasRecentActivity 检查是否有最近的用户活动
func hasRecentActivity(output string, lastCheck time.Time) bool {
	logger.Debug("开始解析 pmset 输出，只看 %v 之后的日志...", lastCheck)
	lineCount := 0
	activityCount := 0
	var latestActivity time.Time
	scanner := bufio.NewScanner(strings.NewReader(output))

	// 限制最多读取1000行
	const maxLines = 1000
	for scanner.Scan() && lineCount < maxLines {
		line := scanner.Text()
		lineCount++
		// 检查是否包含用户活动相关的关键字
		if strings.Contains(line, "UserIsActive") {
			activityCount++
			logger.Debug("发现用户活动日志[%d]: %s", activityCount, line)
			// 解析时间戳
			if timestamp, err := parseTimestamp(line); err == nil {
				// 更新最新活动时间
				if timestamp.After(latestActivity) {
					latestActivity = timestamp
				}
				// 只关注上次检查之后的活动
				if timestamp.After(lastCheck) {
					timeSince := time.Since(timestamp)
					logger.Debug("发现新活动，发生于 %v 之前", timeSince)
					return true
				}
			} else {
				logger.Debug("解析时间戳失败: %v", err)
			}
		}
	}

	if lineCount >= maxLines {
		logger.Debug("达到最大行数限制(%d行)，停止解析", maxLines)
	}

	logger.Debug("完成日志解析，共 %d 行，发现 %d 条活动记录，最新活动时间: %v", lineCount, activityCount, latestActivity)
	return false
}

// parseTimestamp 解析日志中的时间戳
func parseTimestamp(line string) (time.Time, error) {
	// 日志格式示例: "2024-12-29 18:26:13 +0800"
	parts := strings.SplitN(line, " ", 3)
	if len(parts) >= 2 {
		dateStr := parts[0] + " " + parts[1]
		logger.Debug("尝试解析时间戳: %s", dateStr)
		return time.ParseInLocation("2006-01-02 15:04:05", dateStr, time.Local)
	}
	return time.Time{}, fmt.Errorf("invalid timestamp format: %s", line)
}

// Stop 停止监听
func (m *darwinDeviceMonitor) Stop() error {
	close(m.done)
	return nil
}
