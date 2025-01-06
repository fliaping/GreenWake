package wakepacket

import (
	"context"
	"net"
	"time"

	"greenwake-guard/pkg/logger"
	"greenwake-guard/service/wakelock"
)

type Service struct {
	port        int
	wolTimeout  time.Duration
	wakeLockSvc *wakelock.Service
	conn        *net.UDPConn
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewService(port int, wolTimeout time.Duration, wakeLockSvc *wakelock.Service) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		port:        port,
		wolTimeout:  wolTimeout,
		wakeLockSvc: wakeLockSvc,
		ctx:         ctx,
		cancel:      cancel,
	}
}

func (s *Service) Start() error {
	addr := &net.UDPAddr{
		Port: s.port,
		IP:   net.IPv4zero,
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	s.conn = conn

	go s.listen()
	return nil
}

func (s *Service) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.conn != nil {
		// 在关闭连接前设置一个较短的超时时间
		s.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		s.conn.Close()
		s.conn = nil
	}
	logger.Debug("唤醒包服务已停止")
}

func (s *Service) listen() {
	buffer := make([]byte, 1024)
	readTimeout := time.Second

	defer func() {
		if r := recover(); r != nil {
			logger.Error("唤醒包监听发生panic: %v", r)
		}
	}()

	for {
		select {
		case <-s.ctx.Done():
			logger.Debug("唤醒包监听收到停止信号")
			return
		default:
			if s.conn == nil {
				logger.Debug("UDP连接已关闭，停止监听")
				return
			}

			err := s.conn.SetReadDeadline(time.Now().Add(readTimeout))
			if err != nil {
				if s.ctx.Err() != nil {
					// 如果上下文已取消，说明是正常退出，不记录错误
					return
				}
				logger.Error("设置读取超时失败: %v", err)
				continue
			}

			n, remoteAddr, err := s.conn.ReadFromUDP(buffer)
			if err != nil {
				if s.ctx.Err() != nil {
					// 如果上下文已取消，说明是正常退出，不记录错误
					return
				}
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
					logger.Debug("UDP连接已关闭")
					return
				}
				logger.Error("读取UDP数据失败: %v", err)
				continue
			}

			// 检查是否是唤醒包
			if isWakePacket(buffer[:n]) {
				// 使用新的 HandleWakePacket 方法处理唤醒包
				s.handleWakePacket(remoteAddr.String())
			}
		}
	}
}

func isWakePacket(data []byte) bool {
	// 简单实现：检查数据长度是否为 102 字节（标准魔术包长度）
	if len(data) != 102 {
		return false
	}

	// 检查前 6 个字节是否都是 0xFF（魔术包的同步流）
	for i := 0; i < 6; i++ {
		if data[i] != 0xFF {
			return false
		}
	}

	return true
}

func (s *Service) handleWakePacket(sourceIP string) {
	// 记录唤醒包信息
	logger.Debug("收到唤醒包，来源: %s, 超时时间: %v", sourceIP, s.wolTimeout)

	// 通知唤醒锁服务
	s.wakeLockSvc.HandleWakePacket(sourceIP)
}
