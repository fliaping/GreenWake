package wakelock

import (
	"fmt"
	"sync/atomic"
	"time"

	"awake/pkg/logger"
	"awake/service/wakeevent"
)

type Strategy string
type SleepMode string

const (
	StrategyWolWake   Strategy  = "wol_wake"
	StrategyPermanent Strategy  = "permanent" // 永久唤醒
	StrategyTimed     Strategy  = "timed"     // 定时唤醒
	SleepModeSystem   SleepMode = "system"    // 系统控制
	SleepModeProgram  SleepMode = "program"   // 程序控制
)

// Service 唤醒锁服务
type Service struct {
	strategy               Strategy            // 当前策略
	sleepMode              SleepMode           // 睡眠模式（系统控制/程序控制）
	isTemporaryWake        int32               // 是否处于临时唤醒状态（收到唤醒包或计时唤醒）
	programSleepDelay      int                 // 程序控制睡眠模式下等待睡眠时间（秒）
	wolTimeoutSecs         int                 // WOL 唤醒超时时间（秒）
	validEvents            []string            // 有效的唤醒事件类型
	sleepTimer             *time.Timer         // 睡眠定时器
	timerStartTime         time.Time           // 定时器启动时间
	done                   chan struct{}       // 用于停止检查定时器
	lastWakeEvent          wakeevent.EventType // 最后一次唤醒事件类型
	remainingTime          int                 // 剩余时间（秒）
	duration               time.Duration       // 持续时间
	updateCallback         func()              // 状态更新回调函数
	strategyChangeCallback func(Strategy)      // 策略变更回调函数
	lock                   Lock                // 系统唤醒锁
}

// NewService 创建新的服务实例
func NewService(lock Lock) *Service {
	s := &Service{
		strategy:          StrategyWolWake,
		sleepMode:         SleepModeProgram,                     // 默认使用程序控制模式
		programSleepDelay: 60,                                   // 默认60秒
		wolTimeoutSecs:    300,                                  // 默认5分钟
		validEvents:       []string{"wol", "keyboard", "mouse"}, // 默认有效事件
		done:              make(chan struct{}),
		lock:              lock,
	}

	// 启动检查定时器
	go s.runCheckTimer()

	logger.Info("创建新的唤醒锁服务 - 初始状态：策略=%v, 睡眠模式=%v", s.strategy, s.sleepMode)
	return s
}

// runCheckTimer 运行检查定时器
func (s *Service) runCheckTimer() {
	logger.Info("启动睡眠定时器状态检查器")
	ticker := time.NewTicker(3 * time.Second)

	defer ticker.Stop()

	// 立即执行一次检查
	s.checkStatus()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.checkStatus()
		}
	}
}

// checkStatus 检查并打印状态
func (s *Service) checkStatus() {

	// 计算剩余时间
	var remainingStr string
	if s.sleepTimer != nil && !s.timerStartTime.IsZero() {
		remaining := s.programSleepDelay - int(time.Since(s.timerStartTime).Seconds())
		if remaining > 0 {
			remainingStr = fmt.Sprintf("剩余%d秒", remaining)
		} else {
			remainingStr = "即将执行"
		}
	} else {
		remainingStr = "无定时器"
	}

	// 获取最后一次唤醒事件类型的字符串表示
	var eventTypeStr string
	if s.lastWakeEvent == "" {
		eventTypeStr = "未知"
	} else {
		eventTypeStr = string(s.lastWakeEvent)
	}

	// 始终输出定时器状态日志
	logger.Debug("检查睡眠定时器状态 - 当前状态：策略=%v, 临时唤醒=%v, 睡眠模式=%v, 定时器=%v, %s, 唤醒事件=%v",
		s.strategy, atomic.LoadInt32(&s.isTemporaryWake) == 1, s.sleepMode, s.sleepTimer != nil, remainingStr, eventTypeStr)

	// 检查是否需要启动睡眠定时器：
	// 1. 策略是 WOL 唤醒
	// 2. 不在临时唤醒状态（说明已经超过 WOL 超时时间）
	// 3. 是程序控制模式
	// 4. 没有运行中的睡眠定时器
	if s.strategy == StrategyWolWake && atomic.LoadInt32(&s.isTemporaryWake) == 0 && s.sleepMode == SleepModeProgram && s.sleepTimer == nil {
		logger.Info("满足启动睡眠定时器的条件，启动定时器，延迟时间：%d秒", s.programSleepDelay)
		s.startSleepTimer()
	}
}

// HandleWakeEvent 处理唤醒事件
func (s *Service) HandleWakeEvent(event wakeevent.Event) {
	logger.Debug("处理唤醒事件")

	// 检查是否是有效的唤醒事件类型
	eventType := string(event.Type)
	isValidEvent := false
	for _, validType := range s.validEvents {
		if eventType == validType {
			isValidEvent = true
			break
		}
	}

	if !isValidEvent {
		logger.Debug("忽略无效的唤醒事件类型: %s", eventType)
		return
	}

	// 记录详细的唤醒事件信息
	logger.Info("收到唤醒事件 - 类型：%s，来源：%s，时间：%v", event.Type.String(), event.Source, event.Timestamp)
	logger.Debug("唤醒事件详情 - 当前策略：%s，睡眠模式：%s，临时唤醒：%v", s.strategy, s.sleepMode, atomic.LoadInt32(&s.isTemporaryWake) == 1)

	// 取消运行中的睡眠定时器
	if s.sleepTimer != nil {
		logger.Debug("取消正在运行的睡眠定时器")
		s.cancelSleepTimer()
	}

	// 设置临时唤醒状态和事件类型
	atomic.StoreInt32(&s.isTemporaryWake, 1)
	s.lastWakeEvent = event.Type

	// 如果是程序控制模式，启动 WOL 超时定时器
	wolTimeoutSecs := s.wolTimeoutSecs

	if s.sleepMode == SleepModeProgram {
		logger.Debug("程序控制模式，等待 %d 秒后重置临时唤醒状态", wolTimeoutSecs)
		time.AfterFunc(time.Duration(wolTimeoutSecs)*time.Second, func() {
			// 重置临时唤醒状态，让检查器可以启动睡眠定时器
			if atomic.LoadInt32(&s.isTemporaryWake) == 1 {
				logger.Debug("WOL 超时，重置临时唤醒状态")
				atomic.StoreInt32(&s.isTemporaryWake, 0)
			}
		})
	} else {
		logger.Debug("系统控制模式，不启动睡眠定时器")
	}

	if s.updateCallback != nil {
		logger.Debug("触发状态更新回调")
		s.updateCallback()
	}
}

// startSleepTimer 启动睡眠定时器
func (s *Service) startSleepTimer() {
	// 如果已经有定时器在运行，先取消它
	s.cancelSleepTimer()

	// 记录定时器启动时间
	s.timerStartTime = time.Now()

	// 创建新的定时器
	s.sleepTimer = time.AfterFunc(time.Duration(s.programSleepDelay)*time.Second, func() {
		// 执行睡眠
		if err := s.forceSystemSleep(); err != nil {
			logger.Error("执行系统睡眠失败: %v", err)
		}
	})
}

// cancelSleepTimer 取消睡眠定时器
func (s *Service) cancelSleepTimer() {
	if s.sleepTimer != nil {
		s.sleepTimer.Stop()
		s.sleepTimer = nil
		s.timerStartTime = time.Time{} // 清空定时器启动时间
	}
}

// Stop 停止服务
func (s *Service) Stop() {
	close(s.done)
	s.cancelSleepTimer()
}

// forceSystemSleep 强制系统进入睡眠状态
func (s *Service) forceSystemSleep() error {
	// 释放唤醒锁
	s.lock.Release()
	// 强制系统睡眠
	return s.lock.ForceSleep()
}

// GetStrategy 获取当前策略
func (s *Service) GetStrategy() Strategy {
	return s.strategy
}

// SetStrategy 设置策略
func (s *Service) SetStrategy(strategy Strategy, duration time.Duration) {
	s.strategy = strategy
	if strategy == StrategyTimed {
		s.remainingTime = int(duration.Seconds())
		s.duration = duration
	}
	if s.updateCallback != nil {
		s.updateCallback()
	}
	if s.strategyChangeCallback != nil {
		s.strategyChangeCallback(s.strategy)
	}
}

// GetRemainingTime 获取剩余时间（秒）
func (s *Service) GetRemainingTime() int {
	if s.strategy == StrategyTimed {
		return s.remainingTime
	}
	return 0
}

// FormatRemainingTime 格式化剩余时间
func (s *Service) FormatRemainingTime() string {
	if s.strategy == StrategyTimed {
		hours := s.remainingTime / 3600
		minutes := (s.remainingTime % 3600) / 60
		seconds := s.remainingTime % 60
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return "00:00:00"
}

// HandleWakePacket 处理唤醒包
func (s *Service) HandleWakePacket(source string) {
	s.HandleWakeEvent(wakeevent.Event{
		Type:   wakeevent.EventTypeWOL,
		Source: source,
	})
}

// SetUpdateCallback 设置状态更新回调函数
func (s *Service) SetUpdateCallback(callback func()) {
	s.updateCallback = callback
}

// GetDuration 获取持续时间
func (s *Service) GetDuration() time.Duration {
	return s.duration
}

// SetDuration 设置持续时间
func (s *Service) SetDuration(duration time.Duration) {
	s.duration = duration
}

// SetSleepMode 设置睡眠模式
func (s *Service) SetSleepMode(mode SleepMode) {
	s.sleepMode = mode
	if s.updateCallback != nil {
		s.updateCallback()
	}
}

// GetSleepMode 获取当前睡眠模式
func (s *Service) GetSleepMode() SleepMode {
	return s.sleepMode
}

// InitializeState 初始化服务状态
func (s *Service) InitializeState(strategy Strategy, sleepMode SleepMode, duration time.Duration) {
	logger.Info("初始化服务状态 - 策略：%v，睡眠模式：%v，持续时间：%v", strategy, sleepMode, duration)

	// 设置状态
	s.strategy = strategy
	s.sleepMode = sleepMode
	atomic.StoreInt32(&s.isTemporaryWake, 0)
	s.programSleepDelay = 60 // 默认60秒
	s.duration = duration

	// 取消所有定时器
	s.cancelSleepTimer()

	// 触发回调
	if s.updateCallback != nil {
		s.updateCallback()
	}
	if s.strategyChangeCallback != nil {
		s.strategyChangeCallback(s.strategy)
	}

	logger.Info("服务状态初始化完成 - 当前状态：策略=%v, 睡眠模式=%v", s.strategy, s.sleepMode)
}

// SetStrategyChangeCallback 设置策略变更回调函数
func (s *Service) SetStrategyChangeCallback(callback func(Strategy, SleepMode, time.Duration)) {
	s.strategyChangeCallback = func(strategy Strategy) {
		callback(strategy, s.sleepMode, s.duration)
	}
}

// SetProgramSleepDelay 设置程序控制睡眠模式下的等待睡眠时间（秒）
func (s *Service) SetProgramSleepDelay(delay int) {
	s.programSleepDelay = delay
	if s.updateCallback != nil {
		s.updateCallback()
	}
}

// GetProgramSleepDelay 获取程序控制睡眠模式下的等待睡眠时间（秒）
func (s *Service) GetProgramSleepDelay() int {
	return s.programSleepDelay
}

// SetWolTimeoutSecs 设置 WOL 唤醒超时时间（秒）
func (s *Service) SetWolTimeoutSecs(timeout int) {
	s.wolTimeoutSecs = timeout
	if s.updateCallback != nil {
		s.updateCallback()
	}
}

// GetWolTimeoutSecs 获取 WOL 唤醒超时时间（秒）
func (s *Service) GetWolTimeoutSecs() int {
	return s.wolTimeoutSecs
}

// SetValidEvents 设置有效的唤醒事件类型
func (s *Service) SetValidEvents(events []string) {
	s.validEvents = events
	if s.updateCallback != nil {
		s.updateCallback()
	}
}

// GetValidEvents 获取有效的唤醒事件类型
func (s *Service) GetValidEvents() []string {
	return s.validEvents
}
