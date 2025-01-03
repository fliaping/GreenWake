package wakepacket

import (
	"context"
	"net"
	"time"

	"awake/pkg/logger"
	"awake/service/wakelock"
)

type Service struct {
	port        int
	wolTimeout     time.Duration
	wakeLockSvc *wakelock.Service
	conn        *net.UDPConn
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewService(port int, wolTimeout time.Duration, wakeLockSvc *wakelock.Service) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		port:        port,
		wolTimeout:     wolTimeout,
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
	s.cancel()
	if s.conn != nil {
		s.conn.Close()
	}
}

func (s *Service) listen() {
	buffer := make([]byte, 1024)
	lastPacketTime := time.Now()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			s.conn.SetReadDeadline(time.Now().Add(time.Second))
			n, remoteAddr, err := s.conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// 检查是否超时
					if time.Since(lastPacketTime) > s.wolTimeout {
						// 如果超时且当前策略是唤醒包策略，则释放唤醒锁
						if s.wakeLockSvc.GetStrategy() == wakelock.StrategyWolWake {
							s.wakeLockSvc.SetStrategy(wakelock.StrategyWolWake, 0)
						}
					}
					continue
				}
				logger.Error("读取UDP数据失败: %v", err)
				continue
			}

			// 更新最后收到包的时间
			lastPacketTime = time.Now()

			// 检查是否是唤醒包
			if isWakePacket(buffer[:n]) {
				logger.Info("收到来自 %v 的唤醒包", remoteAddr)
				// 如果当前是唤醒包策略，则获取唤醒锁
				if s.wakeLockSvc.GetStrategy() == wakelock.StrategyWolWake {
					s.wakeLockSvc.SetStrategy(wakelock.StrategyPermanent, 0)
				}
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
