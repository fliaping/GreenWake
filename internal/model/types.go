package model

type PCHostInfo struct {
	Name        string `json:"name"`
	IP          string `json:"ip"`
	MAC         string `json:"mac"`
	MonitorPort int    `json:"monitorPort"`
}

type PCHostStatus struct {
	Name         string `json:"name"`
	IsOnline     bool   `json:"isOnline"`
	KeepAwake    bool   `json:"keepAwake"`
	LastUpdate   string `json:"lastUpdate,omitempty"`
	LastWakeTime string `json:"lastWakeTime,omitempty"`
}

type ClientInfo struct {
	ID         string `json:"id"`
	UserAgent  string `json:"userAgent"`
	IP         string `json:"ip"`
	Port       string `json:"port,omitempty"`
	LastSeen   string `json:"lastSeen"`
	TargetHost string `json:"targetHost"`
}

type ChannelClient struct {
	ID         string `json:"id"`
	IP         string `json:"ip"`
	Port       string `json:"port"`
	Status     string `json:"status"`
	LastActive string `json:"lastActive"`
}

type ForwardChannel struct {
	ID          string           `json:"id"`
	ServicePort int              `json:"servicePort"`
	TargetHost  string           `json:"targetHost"`
	TargetPort  int              `json:"targetPort"`
	Status      string           `json:"status"`
	LastActive  string           `json:"lastActive,omitempty"`
	Clients     []*ChannelClient `json:"clients,omitempty"`
}

type Response struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
