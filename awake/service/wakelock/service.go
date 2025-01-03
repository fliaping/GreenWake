package wakelock

import (
	"fmt"
	"sync"
	"time"

	"awake/pkg/logger"
)

type Strategy string
type SleepMode string
type UpdateCallback func()

const (
	StrategyWolWake   Strategy = "wol_wake"
	StrategyPermanent Strategy = "permanent"
	StrategyTimed     Strategy = "timed"

	SleepModeSystem  SleepMode = "system"
	SleepModeProgram SleepMode = "program"
)

type Service struct {
	mu             sync.RWMutex
	strategy       Strategy
	duration       time.Duration
	startTime      time.Time
	timer          *time.Timer
	updateTimer    *time.Timer
	lock           Lock
	sleepMode      SleepMode
	updateCallback UpdateCallback
}

func NewService(lock Lock) *Service {
	return &Service{
		strategy:  StrategyWolWake,
		lock:      lock,
		sleepMode: SleepModeSystem, // 默认使用系统控制
	}
}

// SetUpdateCallback 设置更新回调函数，用于更新UI显示
func (s *Service) SetUpdateCallback(callback UpdateCallback) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateCallback = callback
}

// FormatRemainingTime 格式化剩余时间显示
func (s *Service) FormatRemainingTime() string {
	remaining := s.GetRemainingTime()
	if remaining <= 0 {
		return ""
	}

	hours := int(remaining.Hours())
	minutes := int(remaining.Minutes()) % 60
	seconds := int(remaining.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf(" (%dh%dm%ds)", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf(" (%dm%ds)", minutes, seconds)
	}
	return fmt.Sprintf(" (%ds)", seconds)
}

// startUpdateTimer 启动更新定时器
func (s *Service) startUpdateTimer() {
	if s.updateTimer != nil {
		s.updateTimer.Stop()
	}

	// 每10秒更新一次显示
	s.updateTimer = time.NewTimer(500 * time.Millisecond)
	go func() {
		for range s.updateTimer.C {
			s.mu.RLock()
			if s.strategy != StrategyTimed {
				s.updateTimer.Stop()
				s.updateTimer = nil
				s.mu.RUnlock()
				return
			}
			callback := s.updateCallback
			s.mu.RUnlock()

			if callback != nil {
				callback()
			}

			s.mu.Lock()
			if s.strategy == StrategyTimed {
				s.updateTimer.Reset(10 * time.Second)
			}
			s.mu.Unlock()
		}
	}()
}

// SetStrategy 设置唤醒策略
func (s *Service) SetStrategy(strategy Strategy, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果策略没有改变，且不是计时策略，则不需要更新
	if s.strategy == strategy && strategy != StrategyTimed {
		return
	}

	// 停止之前的计时器
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if s.updateTimer != nil {
		s.updateTimer.Stop()
		s.updateTimer = nil
	}

	oldStrategy := s.strategy
	s.strategy = strategy

	// 对于计时策略，验证持续时间的有效性
	if strategy == StrategyTimed {
		if duration <= 0 {
			logger.Error("计时唤醒持续时间无效：%v", duration)
			s.strategy = oldStrategy
			return
		}
		s.duration = duration
		s.startTime = time.Now()
		logger.Debug("设置计时策略 - 开始时间: %v, 持续时间: %v",
			s.startTime.Format("15:04:05"), duration)
	}

	// 根据策略设置唤醒锁
	switch strategy {
	case StrategyWolWake:
		s.lock.Release()
		if oldStrategy != StrategyWolWake {
			logger.Info("切换到唤醒包策略，等待唤醒包...")
		}

	case StrategyPermanent:
		if oldStrategy == StrategyWolWake {
			logger.Info("收到唤醒包，切换到永久唤醒策略")
		} else {
			logger.Info("切换到永久唤醒策略")
		}
		s.lock.Acquire()

	case StrategyTimed:
		s.lock.Acquire()
		// 创建新的计时器，使用 AfterFunc 确保在计时结束时执行回调
		s.timer = time.AfterFunc(duration, func() {
			s.mu.Lock()
			defer s.mu.Unlock()

			// 检查策略是否仍然是计时策略
			if s.strategy != StrategyTimed {
				return
			}

			// 切换回唤醒包策略
			s.strategy = StrategyWolWake
			s.lock.Release()
			s.timer = nil
			if s.updateCallback != nil {
				s.updateCallback()
			}
			logger.Info("计时唤醒结束（持续时间：%v），切换到唤醒包策略", s.duration)
		})
		// 启动更新定时器
		s.startUpdateTimer()
		logger.Info("切换到计时唤醒策略，持续时间：%v", duration)
	}
}

// GetStrategy 获取当前唤醒策略
func (s *Service) GetStrategy() Strategy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.strategy
}

// GetDuration 获取计时唤醒的持续时间
func (s *Service) GetDuration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.duration
}

// SetSleepMode 设置睡眠模式
func (s *Service) SetSleepMode(mode SleepMode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sleepMode = mode
	logger.Info("切换到%s睡眠模式", mode)
}

// GetSleepMode 获取当前睡眠模式
func (s *Service) GetSleepMode() SleepMode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sleepMode
}

// ForceSystemSleep 强制系统进入睡眠状态
func (s *Service) ForceSystemSleep() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 停止之前的计时器
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}

	// 释放唤醒锁
	s.lock.Release()
	s.strategy = StrategyWolWake

	// 根据睡眠模式选择睡眠方式
	if s.sleepMode == SleepModeSystem {
		logger.Info("使用系统控制进入睡眠状态")
		return s.lock.ForceSleep()
	} else {
		logger.Info("使用程序控制进入睡眠状态")
		s.lock.Release()
		return nil
	}
}

// GetRemainingTime 获取计时唤醒剩余时间
func (s *Service) GetRemainingTime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.strategy != StrategyTimed || s.timer == nil {
		return 0
	}

	// 计算剩余时间
	elapsed := time.Since(s.startTime)
	if elapsed > s.duration {
		return 0
	}
	remaining := s.duration - elapsed
	logger.Debug("计算剩余时间 - 开始时间: %v, 持续时间: %v, 已过时间: %v, 剩余时间: %v",
		s.startTime.Format("15:04:05"), s.duration, elapsed, remaining)
	return remaining
}
