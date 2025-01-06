//go:build windows
// +build windows

package wakeevent

import (
	"atomic"
	"greenwake-guard/config"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"greenwake-guard/pkg/logger"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procSetWindowsHookEx = user32.NewProc("SetWindowsHookExW")
	procGetMessage       = user32.NewProc("GetMessageW")
	procCallNextHookEx   = user32.NewProc("CallNextHookEx")
)

const (
	WH_KEYBOARD_LL = 13
	WH_MOUSE_LL    = 14
)

// windowsDeviceMonitor Windows设备监听器
type windowsDeviceMonitor struct {
	handler Handler
	config  *config.Config
	done    chan struct{}
	// 添加监控状态标志
	monitoringKeyboard atomic.Bool
	monitoringMouse    atomic.Bool
	mu                 sync.Mutex
}

// newPlatformDeviceMonitor 创建Windows平台的设备监听器
func newPlatformDeviceMonitor(handler Handler) DeviceMonitor {
	return &windowsDeviceMonitor{
		handler: handler,
		done:    make(chan struct{}),
	}
}

// Start 启动监听
func (m *windowsDeviceMonitor) Start() error {
	// 启动键盘监听
	go m.startKeyboardHook()
	// 启动鼠标监听
	go m.startMouseHook()
	return nil
}

func (m *windowsDeviceMonitor) UpdateConfig(config *Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = config

	// 处理键盘监控
	if config.IsEventTypeValid(string(EventTypeDevice)) {
		if !m.monitoringKeyboard.Load() {
			m.monitoringKeyboard.Store(true)
			go m.startKeyboardHook()
		}
	} else {
		m.monitoringKeyboard.Store(false)
	}

	// 处理鼠标监控
	if config.IsEventTypeValid(string(EventTypeDevice)) {
		if !m.monitoringMouse.Load() {
			m.monitoringMouse.Store(true)
			go m.startMouseHook()
		}
	} else {
		m.monitoringMouse.Store(false)
	}
}

func (m *windowsDeviceMonitor) startKeyboardHook() {
	hook, err := setWindowsHookEx(WH_KEYBOARD_LL, func(code int, wparam, lparam uintptr) uintptr {
		m.handler.HandleWakeEvent(Event{
			Type:      EventTypeDevice,
			Timestamp: time.Now(),
			Source:    "keyboard",
		})
		return callNextHookEx(0, code, wparam, lparam)
	})
	if err != nil {
		logger.Error("设置键盘钩子失败: %v", err)
		return
	}
	defer syscall.UnhookWindowsHookEx(hook)

	var msg struct {
		HWND   uintptr
		Msg    uint32
		WParam uintptr
		LParam uintptr
		Time   uint32
		Point  struct{ X, Y int32 }
	}

	for m.monitoringKeyboard.Load() {
		select {
		case <-m.done:
			return
		default:
			procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		}
	}
}

func (m *windowsDeviceMonitor) startMouseHook() {
	hook, err := setWindowsHookEx(WH_MOUSE_LL, func(code int, wparam, lparam uintptr) uintptr {
		m.handler.HandleWakeEvent(Event{
			Type:      EventTypeDevice,
			Timestamp: time.Now(),
			Source:    "mouse",
		})
		return callNextHookEx(0, code, wparam, lparam)
	})
	if err != nil {
		logger.Error("设置鼠标钩子失败: %v", err)
		return
	}
	defer syscall.UnhookWindowsHookEx(hook)

	var msg struct {
		HWND   uintptr
		Msg    uint32
		WParam uintptr
		LParam uintptr
		Time   uint32
		Point  struct{ X, Y int32 }
	}

	for m.monitoringMouse.Load() {
		select {
		case <-m.done:
			return
		default:
			procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		}
	}
}

// Stop 停止监听
func (m *windowsDeviceMonitor) Stop() error {
	close(m.done)
	return nil
}

func setWindowsHookEx(hookType int, callback func(int, uintptr, uintptr) uintptr) (syscall.Handle, error) {
	mod := syscall.NewLazyDLL("user32.dll")
	proc := mod.NewProc("SetWindowsHookExW")
	handle, _, err := proc.Call(
		uintptr(hookType),
		syscall.NewCallback(callback),
		0,
		0,
	)
	if handle == 0 {
		return 0, err
	}
	return syscall.Handle(handle), nil
}

func callNextHookEx(hhk syscall.Handle, code int, wparam, lparam uintptr) uintptr {
	ret, _, _ := procCallNextHookEx.Call(
		uintptr(hhk),
		uintptr(code),
		wparam,
		lparam,
	)
	return ret
}