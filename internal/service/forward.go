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
	channelClients sync.Map // key: channelId, value: *sync.Map[clientId]*ChannelClient
	activeCount    int64    // 全局活跃连接计数
	countMu        sync.Mutex
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

	// 增加活跃连接计数
	s.countMu.Lock()
	s.activeCount++
	channel.ActiveCount = int(s.activeCount)
	s.countMu.Unlock()

	channel.Status = "active"
	channel.LastActive = time.Now().Format(time.RFC3339)
	s.channels.Store(channel.ServicePort, channel)

	// 获取通道的客户端映射
	clientsMap, _ := s.channelClients.LoadOrStore(channel.ID, &sync.Map{})
	clientsMap.(*sync.Map).Store(clientId, clientInfo)

	// 在连接结束时清理
	defer func() {
		// 减少活跃连接计数
		s.countMu.Lock()
		s.activeCount--
		count := s.activeCount
		s.countMu.Unlock()

		// 更新通道状态
		if count == 0 {
			channel.Status = "inactive"
		}
		channel.ActiveCount = int(count)
		s.channels.Store(channel.ServicePort, channel)

		// 删除客户端记录
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

	// 获取唤醒间隔时间
	wakeInterval := 120 // 默认120秒
	if cfgHost, exists := s.pcService.cfgHosts[channel.TargetHost]; exists && cfgHost.WakeInterval > 0 {
		wakeInterval = cfgHost.WakeInterval
	}

	// 创建定时唤醒的 ticker
	wakeTicker := time.NewTicker(time.Duration(wakeInterval) * time.Second)
	defer wakeTicker.Stop()

	// 启动定时唤醒协程
	stopWake := make(chan struct{})
	defer close(stopWake)

	go func() {
		for {
			select {
			case <-wakeTicker.C:
				// 检查主机是否在线
				if host, exists := s.pcService.hosts[channel.TargetHost]; exists {
					// 通道活跃期间持续发送唤醒包，保持主机在线
					log.Printf("保持主机唤醒: %s", channel.TargetHost)
					s.pcService.sendWakePacket(host)
				}
			case <-stopWake:
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

	isOnline := checkHostOnline(host.IP, host.MonitorPort)
	if !isOnline {
		cfgHost, exists := s.pcService.cfgHosts[channel.TargetHost]
		retryCount := 1 // 默认重试1次
		if exists && cfgHost.RetryCount > 0 {
			retryCount = cfgHost.RetryCount
		}

		// 重试循环
		for retry := 0; retry <= retryCount; retry++ {
			if retry > 0 {
				log.Printf("第%d/%d次重试唤醒主机: %s", retry, retryCount, channel.TargetHost)
			}

			log.Printf("目标主机离线，尝试唤醒: %s [%d -> %s:%d]", channel.TargetHost, channel.ServicePort, host.IP, channel.TargetPort)
			s.pcService.sendWakePacket(host)

			// 获取唤醒超时时间
			wakeTimeout := 10 // 默认30秒
			if exists && cfgHost.WakeTimeout > 0 {
				wakeTimeout = cfgHost.WakeTimeout
			}

			// 等待主机上线
			startTime := time.Now()
			for {
				if checkHostOnline(host.IP, host.MonitorPort) {
					log.Printf("目标主机已上线，开始转发: %s [%d -> %s:%d]", channel.TargetHost, channel.ServicePort, host.IP, channel.TargetPort)
					goto Connected
				}

				if time.Since(startTime) > time.Duration(wakeTimeout)*time.Second {
					log.Printf("等待主机上线超时（%d秒）: %s [%d -> %s:%d]", wakeTimeout, channel.TargetHost, channel.ServicePort, host.IP, channel.TargetPort)
					break
				}

				time.Sleep(100 * time.Millisecond)
			}

			// 如果是最后一次重试且失败
			if retry == retryCount {
				log.Printf("已重试%d次，主机仍未上线，放弃连接: %s", retryCount, channel.TargetHost)
				return
			}
		}
	}

Connected:
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

func (s *ForwardService) aggregateClients(channelId string) []*model.AggregatedClient {
	clientMap := make(map[string]*model.AggregatedClient)

	if clientsMap, ok := s.channelClients.Load(channelId); ok {
		clientsMap.(*sync.Map).Range(func(_, v interface{}) bool {
			if client, ok := v.(*model.ChannelClient); ok {
				if agg, exists := clientMap[client.IP]; exists {
					agg.Ports = append(agg.Ports, client.Port)
					if client.LastActive > agg.LastActive {
						agg.LastActive = client.LastActive
					}
				} else {
					clientMap[client.IP] = &model.AggregatedClient{
						IP:         client.IP,
						Ports:      []string{client.Port},
						Status:     client.Status,
						LastActive: client.LastActive,
					}
				}
			}
			return true
		})
	}

	// 转换为数组
	aggregatedClients := make([]*model.AggregatedClient, 0, len(clientMap))
	for _, client := range clientMap {
		aggregatedClients = append(aggregatedClients, client)
	}
	return aggregatedClients
}

func (s *ForwardService) GetHostChannels(hostName string) []*model.ForwardChannel {
	var channels []*model.ForwardChannel
	s.channels.Range(func(_, value interface{}) bool {
		if channel, ok := value.(*model.ForwardChannel); ok {
			if channel.TargetHost == hostName {
				channel.Clients = s.aggregateClients(channel.ID)
				// 更新活跃连接数
				s.countMu.Lock()
				channel.ActiveCount = int(s.activeCount)
				s.countMu.Unlock()
				channels = append(channels, channel)
			}
		}
		return true
	})
	return channels
}

func (s *ForwardService) logError(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}

func (s *ForwardService) logInfo(format string, v ...interface{}) {
	log.Printf("[INFO] "+format, v...)
}
