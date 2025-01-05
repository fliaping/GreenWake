import axios, { AxiosError } from 'axios';

const PAGE_ID = crypto.randomUUID();

const api = axios.create({
  baseURL: '/api',
  headers: {
    'X-Page-ID': PAGE_ID
  }
});

export interface APIError {
  error?: string;
}

export interface APIResponse<T> {
  success: boolean;
  error?: string;
  data: T;
}

// 修改 axios 错误处理
api.interceptors.response.use(
  response => response,
  (error: AxiosError<APIError>) => {
    return Promise.reject(error);
  }
);

const KEEP_AWAKE_KEY = 'pc-keep-awake-settings';

export const pcStatusApi = {
  getHosts: () => api.get<{ success: boolean; data: PCHostInfo[] }>('/pc/hosts')
    .then(res => res.data),
  
  getConfig: () => api.get<{ success: boolean; data: { refreshInterval: number } }>('/pc/config')
    .then(res => res.data.data),

  getHostStatus: async (hostName: string, keepAwake?: boolean) => {
    const url = keepAwake ? 
      `/pc/${hostName}/status?keepAwake=true` : 
      `/pc/${hostName}/status`;
    const res = await api.get<{ success: boolean; data: PCHostStatus }>(url);
    return res.data.data;
  },

  getHostClients: (hostName: string) => 
    api.get<{ success: boolean; data: ClientInfo[] }>(`/pc/${hostName}/client_info`)
      .then(res => res.data.data),

  getHostChannels: (hostName: string) => 
    api.get<{ success: boolean; data: ForwardChannel[] }>(`/pc/${hostName}/forward_channels`)
      .then(res => res.data.data),

  getKeepAwakeSettings: (): Record<string, boolean> => {
    try {
      return JSON.parse(localStorage.getItem(KEEP_AWAKE_KEY) || '{}');
    } catch {
      return {};
    }
  },

  setLocalKeepAwake: (hostName: string, keepAwake: boolean) => {
    const settings = pcStatusApi.getKeepAwakeSettings();
    settings[hostName] = keepAwake;
    localStorage.setItem(KEEP_AWAKE_KEY, JSON.stringify(settings));
  }
};

export const clientInfoApi = {
  getInfo: () => api.get<ClientInfo[]>('/client/info').then(res => res.data)
};

export const forwardChannelApi = {
  getChannels: () => api.get<ForwardChannel[]>('/forward/channels').then(res => res.data)
};