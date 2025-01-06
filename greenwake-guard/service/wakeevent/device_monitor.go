package wakeevent

import "greenwake-guard/config"

// DeviceMonitor 设备监听器接口
type DeviceMonitor interface {
	Start() error
	Stop() error
	// 添加更新配置的方法
	UpdateConfig(config *config.Config)
}

// platformDeviceMonitor 平台特定的设备监听器实现
type platformDeviceMonitor interface {
	DeviceMonitor
	newPlatformDeviceMonitor(handler Handler) DeviceMonitor
}

// NewDeviceMonitor 创建平台特定的设备监听器
func NewDeviceMonitor(handler Handler) DeviceMonitor {
	return newPlatformDeviceMonitor(handler)
}
