package service

import (
	"log"
	"my-wol/internal/model"
	"sync"
	"time"
)

type ClientService struct {
	clients sync.Map // key: clientId, value: *model.ClientInfo
	cleaner *time.Ticker
}

func NewClientService() *ClientService {
	s := &ClientService{
		cleaner: time.NewTicker(40 * time.Second), // 改为40秒清理一次
	}

	// 启动清理协程
	go s.cleanInactiveClients()

	return s
}

func (s *ClientService) cleanInactiveClients() {
	for range s.cleaner.C {
		now := time.Now()
		s.clients.Range(func(key, value interface{}) bool {
			if client, ok := value.(*model.ClientInfo); ok {
				lastSeen, err := time.Parse(time.RFC3339, client.LastSeen)
				if err == nil {
					// 如果超过40秒没有收到该客户端的请求，则删除
					if now.Sub(lastSeen) > 40*time.Second {
						log.Printf("清理不活跃客户端: %s, IP: %s, 最后活跃: %s",
							client.ID, client.IP, client.LastSeen)
						s.clients.Delete(key)
					}
				}
			}
			return true
		})
	}
}

func (s *ClientService) Close() {
	if s.cleaner != nil {
		s.cleaner.Stop()
	}
}

// UpdateClient 更新或添加客户端信息
func (s *ClientService) UpdateClient(id string, userAgent string, ip string, port string, targetHost string) {
	s.clients.Store(id, &model.ClientInfo{
		ID:         id,
		UserAgent:  userAgent,
		IP:         ip,
		Port:       port,
		LastSeen:   time.Now().Format(time.RFC3339),
		TargetHost: targetHost,
	})
}

func (s *ClientService) GetHostClients(hostName string) []*model.ClientInfo {
	var clients []*model.ClientInfo
	s.clients.Range(func(_, value interface{}) bool {
		if client, ok := value.(*model.ClientInfo); ok {
			if client.TargetHost == hostName {
				// 只返回40秒内活跃的客户端
				if lastSeen, err := time.Parse(time.RFC3339, client.LastSeen); err == nil {
					if time.Since(lastSeen) <= 40*time.Second {
						clients = append(clients, client)
					}
				}
			}
		}
		return true
	})
	return clients
}
