package wakeevent

import (
	"net"
	"time"

	"greenwake-guard/pkg/logger"
)

// WOLMonitor WOL包监听器
type WOLMonitor struct {
	port    int
	handler Handler
	conn    *net.UDPConn
	done    chan struct{}
}

// NewWOLMonitor 创建WOL包监听器
func NewWOLMonitor(port int, handler Handler) *WOLMonitor {
	return &WOLMonitor{
		port:    port,
		handler: handler,
		done:    make(chan struct{}),
	}
}

// Start 启动监听
func (m *WOLMonitor) Start() error {
	addr := &net.UDPAddr{Port: m.port}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	m.conn = conn

	go func() {
		buf := make([]byte, 1024)
		for {
			select {
			case <-m.done:
				return
			default:
				m.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
				n, remoteAddr, err := m.conn.ReadFromUDP(buf)
				if err != nil {
					if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
						continue
					}
					logger.Error("读取WOL包失败: %v", err)
					continue
				}

				// 验证是否是有效的WOL包
				if n == 102 || n == 108 {
					m.handler.HandleWakeEvent(Event{
						Type:      EventTypeWOL,
						Timestamp: time.Now(),
						Source:    remoteAddr.String(),
					})
				}
			}
		}
	}()

	return nil
}

// Stop 停止监听
func (m *WOLMonitor) Stop() error {
	close(m.done)
	if m.conn != nil {
		return m.conn.Close()
	}
	return nil
}
