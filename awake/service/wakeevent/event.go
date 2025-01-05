package wakeevent

import "time"

// EventType 唤醒事件类型
type EventType string

const (
	EventTypeWOL      EventType = "wol"      // WOL 唤醒包
	EventTypeKeyboard EventType = "keyboard" // 键盘输入
	EventTypeMouse    EventType = "mouse"    // 鼠标输入
)

// Event 唤醒事件
type Event struct {
	Type      EventType // 事件类型
	Timestamp time.Time // 事件发生时间
	Source    string    // 事件来源（例如：WOL包的源IP，或设备标识符）
}

// String 返回事件类型的字符串表示
func (t EventType) String() string {
	switch t {
	case EventTypeWOL:
		return "WOL包"
	case EventTypeKeyboard:
		return "键盘"
	case EventTypeMouse:
		return "鼠标"
	default:
		return "未知"
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
