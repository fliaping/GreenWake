type Config struct {
	// 程序控制睡眠模式下等待睡眠时间（秒）
	ProgramSleepDelay int `yaml:"program_sleep_delay"`

	// 唤醒包相关配置
	WolWake struct {
		WolPort           int `yaml:"wol_port"`            // 唤醒包监听端口
		WolTimeoutMinutes int `yaml:"wol_timeout_minutes"` // 唤醒包超时时间（分钟）
	} `yaml:"wol_wake"`

	// 自动启动配置
	AutoStart bool `yaml:"auto_start"`

	// 初始配置
	Initial struct {
		Strategy  string `yaml:"strategy"`   // 初始唤醒策略
		SleepMode string `yaml:"sleep_mode"` // 初始睡眠模式
		Duration  string `yaml:"duration"`   // 初始计时时长
	} `yaml:"initial"`
}

// WolWakeConfig WOL唤醒配置
type WolWakeConfig struct {
	WolPort        int    `yaml:"wol_port"`         // WOL端口
	WolTimeoutSecs int    `yaml:"wol_timeout_secs"` // WOL超时时间（秒）
	ValidEvents    string `yaml:"valid_events"`     // 有效的唤醒事件类型，多个类型用逗号分隔
}

// GetValidEvents 获取有效的唤醒事件类型列表
func (c *WolWakeConfig) GetValidEvents() []string {
	if c.ValidEvents == "" {
		return []string{"wol", "keyboard", "mouse"} // 默认值
	}
	events := strings.Split(c.ValidEvents, ",")
	// 去除空格
	for i := range events {
		events[i] = strings.TrimSpace(events[i])
	}
	return events
} 