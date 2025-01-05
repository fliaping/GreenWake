package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"awake/config"
	"awake/pkg/logger"
	"awake/pkg/singleinstance"
	"awake/service/tray"
	"awake/service/wakeevent"
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

	// 获取配置文件路径
	configPath := *configFile
	if configPath == "" {
		configPath = config.GetConfigPath()
	}

	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Error("加载配置失败: %v", err)
		os.Exit(1)
	}

	// 检查是否已有实例在运行
	lock := singleinstance.New("awake")
	if err := lock.TryLock(cfg); err != nil {
		logger.Error("%v", err)
		os.Exit(1)
	}
	defer lock.Release()

	// 创建上下文和取消函数
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建等待组
	var wg sync.WaitGroup

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 创建唤醒锁服务
	wakeLockSvc := wakelock.NewService(wakelock.NewLock())

	// 设置程序控制睡眠模式下等待睡眠时间
	wakeLockSvc.SetProgramSleepDelay(cfg.ProgramSleepDelay)
	// 设置 WOL 超时时间
	wakeLockSvc.SetWolTimeoutSecs(cfg.WolWake.WolTimeoutSecs)
	// 设置有效的唤醒事件类型
	wakeLockSvc.SetValidEvents(cfg.WolWake.GetValidEvents())

	// 创建托盘服务
	traySvc := tray.NewTrayService(wakeLockSvc)

	// 设置配置（这会同时设置初始策略和睡眠模式）
	traySvc.SetConfig(cfg, configPath)

	// 创建唤醒包服务并在后台启动
	wakePacketSvc := wakepacket.NewService(cfg.WolWake.WolPort, time.Duration(cfg.WolWake.WolTimeoutSecs)*time.Second, wakeLockSvc)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := wakePacketSvc.Start(); err != nil {
			if ctx.Err() == nil { // 只有在非正常退出时才记录错误
				logger.Error("启动唤醒包服务失败: %v", err)
				cancel() // 触发其他服务退出
			}
		}
	}()

	// 创建并启动设备监控器
	deviceMonitor := wakeevent.NewDeviceMonitor(wakeLockSvc)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := deviceMonitor.Start(); err != nil {
			if ctx.Err() == nil { // 只有在非正常退出时才记录错误
				logger.Error("启动设备监控器失败: %v", err)
				cancel() // 触发其他服务退出
			}
		}
	}()

	// 启动信号处理
	go func() {
		sig := <-sigChan
		logger.Info("收到信号: %v, 开始优雅退出", sig)
		cancel() // 触发所有服务退出
	}()

	// 启动托盘服务
	go func() {
		traySvc.Start()
		cancel() // 如果托盘服务退出，触发其他服务退出
	}()

	// 等待上下文取消
	<-ctx.Done()
	logger.Info("开始关闭服务...")

	// 按顺序关闭服务
	traySvc.Stop()
	deviceMonitor.Stop()
	wakePacketSvc.Stop()

	// 等待所有 goroutine 完成
	logger.Info("等待服务完全停止...")
	wg.Wait()
	logger.Info("所有服务已停止，程序退出")
}
