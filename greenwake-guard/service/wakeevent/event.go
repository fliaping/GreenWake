package wakeevent

import "time"

// EventType 唤醒事件类型
type EventType string

const (
	EventTypeWOL    EventType = "wol"    // 网络唤醒包
	EventTypeDevice EventType = "device" // 外部设备活动（键盘、鼠标等）
)

// Event 唤醒事件
type Event struct {
	Type      EventType // 事件类型
	Source    string    // 事件来源（例如：网络唤醒包的源IP，或设备标识符）
	Timestamp time.Time // 事件发生时间
}

// String 返回事件类型的字符串表示
func (t EventType) String() string {
	switch t {
	case EventTypeWOL:
		return "网络唤醒包"
	case EventTypeDevice:
		return "外部设备活动"
	default:
		return "未知事件"
	}
}

// Handler 唤醒事件处理器
type Handler interface {
	HandleWakeEvent(event Event)
}

// Monitor 唤醒事件监听器接口
type Monitor interface {
	Start() error
	Stop() error
}
