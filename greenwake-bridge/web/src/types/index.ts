interface PCHostInfo {
  name: string;
  ip: string;
  mac: string;
  monitorPort: number;
}

interface PCHostStatus {
  name: string;
  isOnline: boolean;
  keepAwake: boolean;
  lastUpdate?: string;
  lastWakeTime?: string;
}

interface ClientInfo {
  id: string;
  userAgent: string;
  ip: string;
  port: string;
  lastSeen: string;
  targetHost: string;
}

interface ChannelClient {
  id: string;
  ip: string;
  port: string;
  status: string;
  lastActive: string;
}

interface ForwardChannel {
  id: string;
  servicePort: number;
  targetHost: string;
  targetPort: number;
  status: 'active' | 'inactive';
  lastActive?: string;
  clients?: ChannelClient[];
}

interface ServiceLink {
  id: string;
  name: string;
  url: string;
  description?: string;
  targetHost: string;
} 