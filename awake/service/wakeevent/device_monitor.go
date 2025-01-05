package wakeevent

// DeviceMonitor 设备监听器接口
type DeviceMonitor interface {
	Monitor
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
