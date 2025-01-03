package main

import (
	"flag"
	"os"
	"time"

	"awake/config"
	"awake/pkg/logger"
	"awake/service/tray"
	"awake/service/wakelock"
	"awake/service/wakepacket"
)

func main() {
	// 初始化日志
	if err := logger.Init(); err != nil {
		logger.Error("初始化日志失败: %v", err)
		os.Exit(1)
	}
	defer logger.Close()

	// 解析命令行参数
	configFile := flag.String("config", "", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		logger.Error("加载配置失败: %v", err)
		os.Exit(1)
	}

	// 创建唤醒锁服务
	wakeLockSvc := wakelock.NewService(wakelock.NewLock())

	// 创建唤醒包服务并在后台启动
	wakePacketSvc := wakepacket.NewService(cfg.WolWake.WolPort, time.Duration(cfg.WolWake.WolTimeoutMinutes)*time.Minute, wakeLockSvc)
	go func() {
		if err := wakePacketSvc.Start(); err != nil {
			logger.Error("启动唤醒包服务失败: %v", err)
			os.Exit(1)
		}
	}()

	// 创建托盘服务并在主 goroutine 中运行
	traySvc := tray.NewTrayService(wakeLockSvc)
	traySvc.SetConfig(cfg, *configFile)
	traySvc.Start() // 这会阻塞主 goroutine

	// 清理资源
	wakePacketSvc.Stop()
}
