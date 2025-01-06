package wakelock

import (
	"fmt"
	"greenwake-guard/config"
	"sync/atomic"
	"time"

	"greenwake-guard/pkg/logger"
	"greenwake-guard/service/wakeevent"
)

type Strategy string
type SleepMode string

const (
	StrategyExternalWake Strategy  = "external_wake" // 外部唤醒
	StrategyPermanent    Strategy  = "permanent"     // 永久唤醒
	StrategyTimed        Strategy  = "timed"         // 定时唤醒
	SleepModeSystem      SleepMode = "system"        // 系统控制
	SleepModeProgram     SleepMode = "program"       // 程序控制
)

// Service 唤醒锁服务
type Service struct {
	strategy                Strategy            // 当前策略
	sleepMode               SleepMode           // 睡眠模式（系统控制/程序控制）
	isTemporaryWake         int32               // 是否处于临时唤醒状态（收到唤醒包或计时唤醒）
	programSleepDelay       int                 // 程序控制睡眠模式下等待睡眠时间（秒）
	externalWakeTimeoutSecs int                 // 外部唤醒超时时间（秒）
	validEvents             []string            // 有效的唤醒事件类型
	sleepTimer              *time.Timer         // 睡眠定时器
	timerStartTime          time.Time           // 定时器启动时间
	done                    chan struct{}       // 用于停止检查定时器
	lastWakeEvent           wakeevent.EventType // 最后一次唤醒事件类型
	remainingTime           int                 // 剩余时间（秒）
	duration                time.Duration       // 持续时间
	updateCallback          func()              // 状态更新回调函数
	strategyChangeCallback  func(Strategy)      // 策略变更回调函数
	saveConfigCallback      func() error        // 配置保存回调函数
	lock                    Lock                // 系统唤醒锁
}

// NewService 创建新的服务实例
func NewService(lock Lock, cfg *config.Config) *Service {
	s := &Service{
		strategy:                Strategy(cfg.Strategy),
		sleepMode:               SleepMode(cfg.SleepMode),
		programSleepDelay:       cfg.ProgramSleepDelay,
		externalWakeTimeoutSecs: cfg.ExternalWake.TimeoutSecs,
		validEvents:             cfg.ExternalWake.GetValidEvents(),
		done:                    make(chan struct{}),
		lock:                    lock,
	}

	// 只有在程序控制模式下才启动检查定时器
	if s.sleepMode == SleepModeProgram {
		go s.runCheckTimer()
	}

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
			logger.Debug("检查定时器已停止")
			return
		case <-ticker.C:
			// 如果模式已经切换到系统控制，停止检查定时器
			if s.sleepMode != SleepModeProgram {
				logger.Debug("当前为系统控制模式，停止检查定时器")
				return
			}
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
	// 1. 策略是外部唤醒
	// 2. 不在临时唤醒状态（说明已经超过超时时间）
	// 3. 是程序控制模式
	// 4. 没有运行中的睡眠定时器
	// 5. 不是永久唤醒或计时唤醒模式
	if s.strategy == StrategyExternalWake &&
		atomic.LoadInt32(&s.isTemporaryWake) == 0 &&
		s.sleepMode == SleepModeProgram &&
		s.sleepTimer == nil {
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

	// 获取唤醒锁，阻止系统睡眠
	s.lock.Acquire()

	// 设置临时唤醒状态和事件类型
	atomic.StoreInt32(&s.isTemporaryWake, 1)
	s.lastWakeEvent = event.Type

	// 启动外部唤醒超时定时器
	externalWakeTimeoutSecs := s.externalWakeTimeoutSecs
	logger.Debug("启动外部唤醒超时定时器，等待 %d 秒后重置临时唤醒状态", externalWakeTimeoutSecs)
	time.AfterFunc(time.Duration(externalWakeTimeoutSecs)*time.Second, func() {
		// 重置临时唤醒状态
		if atomic.LoadInt32(&s.isTemporaryWake) == 1 {
			logger.Debug("外部唤醒超时，重置临时唤醒状态")
			atomic.StoreInt32(&s.isTemporaryWake, 0)
			// 释放唤醒锁，允许系统睡眠
			s.lock.Release()

			// 如果是程序控制模式，检查定时器会自动启动睡眠定时器
			if s.sleepMode == SleepModeProgram {
				logger.Debug("程序控制模式，检查定时器将在下次检查时启动睡眠定时器")
			}
		}
	})

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
	// 停止服务时释放唤醒锁
	s.lock.Release()
}

// forceSystemSleep 强制系统进入睡眠状态
func (s *Service) forceSystemSleep() error {
	logger.Info("强制系统睡眠")
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
	// 如果切换到永久唤醒或计时唤醒，取消睡眠定时器
	if strategy == StrategyPermanent || strategy == StrategyTimed {
		s.cancelSleepTimer()
	}

	s.strategy = strategy
	if strategy == StrategyTimed {
		s.remainingTime = int(duration.Seconds())
		s.duration = duration
		// 获取唤醒锁，阻止系统睡眠
		s.lock.Acquire()
		// 启动定时器，到期后释放唤醒锁并启动睡眠定时器
		time.AfterFunc(duration, func() {
			s.lock.Release()
			// 如果是程序控制模式，启动睡眠定时器
			if s.sleepMode == SleepModeProgram {
				s.startSleepTimer()
			}
		})
	} else if strategy == StrategyPermanent {
		// 永久唤醒模式下获取唤醒锁
		s.lock.Acquire()
	} else {
		// 其他模式下释放唤醒锁
		s.lock.Release()
	}

	if s.updateCallback != nil {
		s.updateCallback()
	}
	if s.strategyChangeCallback != nil {
		s.strategyChangeCallback(s.strategy)
	}
	// 保存配置
	if s.saveConfigCallback != nil {
		if err := s.saveConfigCallback(); err != nil {
			logger.Error("保存配置失败: %v", err)
		}
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
	if s.sleepMode == mode {
		return
	}

	s.sleepMode = mode

	// 如果切换到系统控制模式，取消已有的睡眠定时器和检查定时器
	if mode == SleepModeSystem {
		logger.Debug("切换到系统控制模式，取消睡眠定时器和检查定时器")
		s.cancelSleepTimer()
		close(s.done)
		s.done = make(chan struct{}) // 重新创建 done channel 以备后用
	} else {
		// 如果切换到程序控制模式，启动检查定时器
		logger.Debug("切换到程序控制模式，启动检查定时器")
		go s.runCheckTimer()
	}

	if s.updateCallback != nil {
		s.updateCallback()
	}
	// 保存配置
	if s.saveConfigCallback != nil {
		if err := s.saveConfigCallback(); err != nil {
			logger.Error("保存配置失败: %v", err)
		}
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
	s.duration = duration

	// 取消所有定时器
	s.cancelSleepTimer()

	// 根据策略设置唤醒锁状态
	if strategy == StrategyPermanent || strategy == StrategyTimed {
		s.lock.Acquire()
		if strategy == StrategyTimed {
			// 启动定时器，到期后释放唤醒锁
			time.AfterFunc(duration, func() {
				s.lock.Release()
			})
		}
	} else {
		s.lock.Release()
	}

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
	if delay < 30 {
		logger.Info("程序控制睡眠模式下等待睡眠时间(%d秒)小于最小值，设置为30秒", delay)
		delay = 30
	}
	s.programSleepDelay = delay
	if s.updateCallback != nil {
		s.updateCallback()
	}
}

// GetProgramSleepDelay 获取程序控制睡眠模式下的等待睡眠时间（秒）
func (s *Service) GetProgramSleepDelay() int {
	return s.programSleepDelay
}

// SetTimeoutSecs 设置外部唤醒超时时间（秒）
func (s *Service) SetTimeoutSecs(timeout int) {
	if timeout < 30 {
		logger.Info("外部唤醒超时时间(%d秒)小于最小值，设置为30秒", timeout)
		timeout = 30
	}
	s.externalWakeTimeoutSecs = timeout
	if s.updateCallback != nil {
		s.updateCallback()
	}
}

// GetTimeoutSecs 获取外部唤醒超时时间（秒）
func (s *Service) GetTimeoutSecs() int {
	return s.externalWakeTimeoutSecs
}

// SetValidEvents 设置有效的唤醒事件类型
func (s *Service) SetValidEvents(events []string) {
	s.validEvents = events
	if s.updateCallback != nil {
		s.updateCallback()
	}
	// 保存配置
	if s.saveConfigCallback != nil {
		if err := s.saveConfigCallback(); err != nil {
			logger.Error("保存配置失败: %v", err)
		}
	}
}

// GetValidEvents 获取有效的唤醒事件类型
func (s *Service) GetValidEvents() []string {
	return s.validEvents
}

// SetSaveConfigCallback 设置配置保存回调函数
func (s *Service) SetSaveConfigCallback(callback func() error) {
	s.saveConfigCallback = callback
}
