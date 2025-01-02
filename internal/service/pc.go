package service

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"my-wol/internal/config"
	"my-wol/internal/model"

	"github.com/sabhiram/go-wol/wol"
)

type PCService struct {
	cfg    *config.Config
	hosts  map[string]*model.PCHostInfo
	status sync.Map // key: hostName, value: *model.PCHostStatus
	wol    sync.Map // key: hostName, value: time.Time (上次唤醒时间)
}

func NewPCService(cfg *config.Config) *PCService {
	s := &PCService{
		cfg:   cfg,
		hosts: make(map[string]*model.PCHostInfo),
	}

	// 初始化主机信息
	for _, host := range cfg.Hosts {
		s.hosts[host.Name] = &model.PCHostInfo{
			Name:        host.Name,
			IP:          host.IP,
			MAC:         host.MAC,
			MonitorPort: host.MonitorPort,
		}
	}

	return s
}

func (s *PCService) GetHosts() []*model.PCHostInfo {
	hosts := make([]*model.PCHostInfo, 0, len(s.hosts))
	for _, host := range s.hosts {
		hosts = append(hosts, host)
	}
	return hosts
}

func (s *PCService) GetHostStatus(hostName string, keepAwake bool) (*model.PCHostStatus, error) {
	host, exists := s.hosts[hostName]
	if !exists {
		return nil, fmt.Errorf("host not found: %s", hostName)
	}

	// 检查主机在线状态
	isOnline := checkHostOnline(host.IP, host.MonitorPort)

	status := &model.PCHostStatus{
		Name:       hostName,
		IsOnline:   isOnline,
		KeepAwake:  keepAwake,
		LastUpdate: time.Now().Format(time.RFC3339),
	}

	// 获取最后唤醒时间
	if lastWake, ok := s.wol.Load(hostName); ok {
		status.LastWakeTime = lastWake.(time.Time).Format(time.RFC3339)
	}

	// 处理唤醒逻辑
	if keepAwake {
		if lastWake, ok := s.wol.Load(hostName); ok {
			// 如果有上次唤醒记录，检查是否需要再次唤醒
			if time.Since(lastWake.(time.Time)) > 5*time.Minute {
				// 超过5分钟，直接发送唤醒包
				go s.sendWakePacket(host)
			}
		} else {
			// 第一次唤醒请求，直接发送唤醒包
			go s.sendWakePacket(host)
		}
	}

	// 存储状态
	s.status.Store(hostName, status)

	return status, nil
}

func (s *PCService) sendWakePacket(host *model.PCHostInfo) {
	log.Printf("发送唤醒包到 %s (MAC: %s)", host.Name, host.MAC)

	mp, err := wol.New(host.MAC)
	if err != nil {
		log.Printf("创建唤醒包失败: %v", err)
		return
	}

	bcastAddr := "255.255.255.255:9"
	udpAddr, err := net.ResolveUDPAddr("udp", bcastAddr)
	if err != nil {
		log.Printf("解析广播地址失败: %v", err)
		return
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		log.Printf("创建UDP连接失败: %v", err)
		return
	}
	defer conn.Close()

	bs, err := mp.Marshal()
	if err != nil {
		log.Printf("序列化唤醒包失败: %v", err)
		return
	}

	n, err := conn.Write(bs)
	if err != nil {
		log.Printf("发送唤醒包失败: %v", err)
		return
	}
	if n != 102 {
		log.Printf("发送的数据长度不正确: %d (应为102字节)", n)
		return
	}

	// 记录唤醒时间
	s.wol.Store(host.Name, time.Now())
	log.Printf("唤醒包发送成功 -> %s", host.Name)
}

func checkHostOnline(ip string, port int) bool {
	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
