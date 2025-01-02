package service

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"my-wol/internal/config"
	"my-wol/internal/model"
)

type ForwardService struct {
	config         *config.Config
	pcService      *PCService
	channels       sync.Map // key: servicePort, value: *model.ForwardChannel
	listeners      map[int]net.Listener
	mu             sync.Mutex
	activeConn     sync.Map // key: servicePort, value: int (活跃连接数)
	channelClients sync.Map // key: channelId, value: map[string]*ChannelClient
	cleaner        *time.Ticker
}

func NewForwardService(cfg *config.Config, pcService *PCService) *ForwardService {
	s := &ForwardService{
		config:    cfg,
		pcService: pcService,
		listeners: make(map[int]net.Listener),
		cleaner:   time.NewTicker(40 * time.Second), // 每40秒清理一次
	}

	// 初始化所有转发通道
	for _, fc := range cfg.Forwards {
		channel := &model.ForwardChannel{
			ID:          fmt.Sprintf("%d-%s:%d", fc.ServicePort, fc.TargetHost, fc.TargetPort),
			ServicePort: fc.ServicePort,
			TargetHost:  fc.TargetHost,
			TargetPort:  fc.TargetPort,
			Status:      "inactive",
		}
		s.channels.Store(fc.ServicePort, channel)
		go s.startForward(channel)
	}

	// 启动清理协程
	go s.cleanInactiveClients()

	return s
}

func (s *ForwardService) cleanInactiveClients() {
	for range s.cleaner.C {
		now := time.Now()
		s.channelClients.Range(func(channelId, value interface{}) bool {
			if clientsMap, ok := value.(*sync.Map); ok {
				clientsMap.Range(func(clientId, v interface{}) bool {
					if client, ok := v.(*model.ChannelClient); ok {
						lastActive, err := time.Parse(time.RFC3339, client.LastActive)
						if err == nil && now.Sub(lastActive) > 40*time.Second {
							log.Printf("清理不活跃转发客户端: 通道=%v, 客户端=%s, IP=%s, 最后活跃=%s",
								channelId, client.ID, client.IP, client.LastActive)
							clientsMap.Delete(clientId)
						}
					}
					return true
				})
			}
			return true
		})
	}
}

func (s *ForwardService) startForward(channel *model.ForwardChannel) {
	s.mu.Lock()
	if _, exists := s.listeners[channel.ServicePort]; exists {
		s.mu.Unlock()
		return
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", channel.ServicePort))
	if err != nil {
		log.Printf("启动转发监听失败 [%d]: %v", channel.ServicePort, err)
		s.mu.Unlock()
		return
	}
	s.listeners[channel.ServicePort] = listener
	s.mu.Unlock()

	log.Printf("启动转发监听 [%d -> %s:%d]", channel.ServicePort, channel.TargetHost, channel.TargetPort)

	for {
		client, err := listener.Accept()
		if err != nil {
			log.Printf("接受连接失败 [%d]: %v", channel.ServicePort, err)
			break
		}

		go s.handleConnection(client, channel)
	}
}

func (s *ForwardService) handleConnection(client net.Conn, channel *model.ForwardChannel) {
	defer client.Close()

	// 生成客户端ID
	clientAddr := client.RemoteAddr().(*net.TCPAddr)
	clientId := fmt.Sprintf("%s:%d", clientAddr.IP.String(), clientAddr.Port)

	// 记录客户端信息
	clientInfo := &model.ChannelClient{
		ID:         clientId,
		IP:         clientAddr.IP.String(),
		Port:       fmt.Sprintf("%d", clientAddr.Port),
		Status:     "active",
		LastActive: time.Now().Format(time.RFC3339),
	}

	// 获取或创建通道的客户端映射
	clientsMap, _ := s.channelClients.LoadOrStore(channel.ID, &sync.Map{})
	clientsMap.(*sync.Map).Store(clientId, clientInfo)

	// 在连接结束时清理客户端信息
	defer func() {
		if cm, ok := s.channelClients.Load(channel.ID); ok {
			cm.(*sync.Map).Delete(clientId)
		}
	}()

	// 启动一个协程持续更新活跃时间
	stopUpdate := make(chan struct{})
	defer close(stopUpdate)

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if cm, ok := s.channelClients.Load(channel.ID); ok {
					if ci, exists := cm.(*sync.Map).Load(clientId); exists {
						if client, ok := ci.(*model.ChannelClient); ok {
							client.LastActive = time.Now().Format(time.RFC3339)
						}
					}
				}
			case <-stopUpdate:
				return
			}
		}
	}()

	// 获取目标主机信息
	host, exists := s.pcService.hosts[channel.TargetHost]
	if !exists {
		log.Printf("目标主机不存在: %s", channel.TargetHost)
		return
	}

	// 检查主机状态并尝试唤醒
	status, err := s.pcService.GetHostStatus(channel.TargetHost, false)
	if err != nil || !status.IsOnline {
		log.Printf("目标主机离线，尝试唤醒: %s [%d -> %s:%d]", channel.TargetHost, channel.ServicePort, host.IP, channel.TargetPort)
		// 发送唤醒包
		s.pcService.sendWakePacket(host)

		// 等待主机上线
		startTime := time.Now()
		for {
			status, err := s.pcService.GetHostStatus(channel.TargetHost, false)
			if err == nil && status.IsOnline {
				log.Printf("目标主机已上线，开始转发: %s [%d -> %s:%d]", channel.TargetHost, channel.ServicePort, host.IP, channel.TargetPort)
				break
			}

			if time.Since(startTime) > 10*time.Second {
				log.Printf("等待主机上线超时（10秒）: %s [%d -> %s:%d]", channel.TargetHost, channel.ServicePort, host.IP, channel.TargetPort)
				return
			}

			time.Sleep(100 * time.Millisecond)
		}
	}

	// 连接目标地址
	target, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host.IP, channel.TargetPort), 5*time.Second)
	if err != nil {
		log.Printf("连接目标失败 [%s:%d]: %v", host.IP, channel.TargetPort, err)
		return
	}
	defer target.Close()

	// 使用 WaitGroup 等待两个方向的数据传输都完成
	var wg sync.WaitGroup
	wg.Add(2)

	// 客户端 -> 目标主机
	go func() {
		defer wg.Done()
		if _, err := io.Copy(target, client); err != nil {
			log.Printf("转发错误 (client->target): %v", err)
		}
		// 通知另一个方向结束
		target.(*net.TCPConn).CloseWrite()
	}()

	// 目标主机 -> 客户端
	go func() {
		defer wg.Done()
		if _, err := io.Copy(client, target); err != nil {
			log.Printf("转发错误 (target->client): %v", err)
		}
		// 通知另一个方向结束
		client.(*net.TCPConn).CloseWrite()
	}()

	// 等待两个方向的数据传输都完成
	wg.Wait()

	// 减少活跃连接计数
	count := 0
	if v, ok := s.activeConn.Load(channel.ServicePort); ok {
		count = v.(int) - 1
		if count > 0 {
			s.activeConn.Store(channel.ServicePort, count)
		} else {
			s.activeConn.Delete(channel.ServicePort)
		}
	}

	// 只有当没有活跃连接时才更新状态为非活跃
	if count == 0 {
		channel.Status = "inactive"
		s.channels.Store(channel.ServicePort, channel)
	}
}

func (s *ForwardService) GetChannels() []*model.ForwardChannel {
	var channels []*model.ForwardChannel
	s.channels.Range(func(_, value interface{}) bool {
		if channel, ok := value.(*model.ForwardChannel); ok {
			channels = append(channels, channel)
		}
		return true
	})
	return channels
}

func (s *ForwardService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cleaner != nil {
		s.cleaner.Stop()
	}

	for _, listener := range s.listeners {
		listener.Close()
	}
}

func (s *ForwardService) GetHostChannels(hostName string) []*model.ForwardChannel {
	var channels []*model.ForwardChannel
	s.channels.Range(func(_, value interface{}) bool {
		if channel, ok := value.(*model.ForwardChannel); ok {
			if channel.TargetHost == hostName {
				// 获取通道的客户端列表
				var clients []*model.ChannelClient
				if clientsMap, ok := s.channelClients.Load(channel.ID); ok {
					clientsMap.(*sync.Map).Range(func(_, v interface{}) bool {
						if client, ok := v.(*model.ChannelClient); ok {
							clients = append(clients, client)
						}
						return true
					})
				}
				channel.Clients = clients
				channels = append(channels, channel)
			}
		}
		return true
	})
	return channels
}
