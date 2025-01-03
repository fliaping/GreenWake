import React, { useEffect, useState } from 'react';
import { Card, Switch, Table, Tag, Typography, Button, Tooltip, Collapse, message } from 'antd';
import { SyncOutlined } from '@ant-design/icons';
import { pcStatusApi, APIError } from '../services';
import { parseUserAgent } from '../utils/userAgent';
import { AxiosError } from 'axios';

const { Title } = Typography;
const { Panel } = Collapse;

const REFRESH_INTERVAL = 30;

const formatDate = (date: string) => {
  return new Date(date).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  }).replace(/\//g, '-');
};

const formatTimeAgo = (dateStr: string) => {
  const date = new Date(dateStr);
  const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000);
  
  if (seconds < 60) {
    return `${seconds}秒前`;
  } else {
    const minutes = Math.floor(seconds / 60);
    return `${minutes}分钟前`;
  }
};

const RemoteControl: React.FC = () => {
  const [hosts, setHosts] = useState<PCHostInfo[]>([]);
  const [hostStatuses, setHostStatuses] = useState<Record<string, PCHostStatus>>({});
  const [hostClients, setHostClients] = useState<Record<string, ClientInfo[]>>({});
  const [hostChannels, setHostChannels] = useState<Record<string, ForwardChannel[]>>({});
  const [countdowns, setCountdowns] = useState<Record<string, number>>({});
  const [refreshingHosts, setRefreshingHosts] = useState<Record<string, boolean>>({});
  const [loadingHosts, setLoadingHosts] = useState<Record<string, boolean>>({});

  // 获取单个主机的状态和相关信息
  const fetchHostData = async (hostName: string, keepAwake?: boolean) => {
    try {
      // 分别发送三个请求
      const statusPromise = pcStatusApi.getHostStatus(hostName, keepAwake)
        .then(status => {
          if (status) {
            setHostStatuses(prev => ({ ...prev, [hostName]: status }));
          }
        });

      const clientsPromise = pcStatusApi.getHostClients(hostName)
        .then(clients => {
          setHostClients(prev => ({ ...prev, [hostName]: clients || [] }));
        });

      const channelsPromise = pcStatusApi.getHostChannels(hostName)
        .then(channels => {
          setHostChannels(prev => ({ ...prev, [hostName]: channels || [] }));
        });

      await Promise.all([statusPromise, clientsPromise, channelsPromise]);
      setCountdowns(prev => ({ ...prev, [hostName]: REFRESH_INTERVAL }));
    } catch (error) {
      console.error(`获取主机 ${hostName} 数据失败:`, error);
    } finally {
      setRefreshingHosts(prev => ({ ...prev, [hostName]: false }));
      setLoadingHosts(prev => ({ ...prev, [hostName]: false }));
    }
  };

  // 初始加载主机列表
  useEffect(() => {
    const loadHosts = async () => {
      try {
        const response = await pcStatusApi.getHosts();
        const hostsData = response.data || [];
        setHosts(hostsData);
        
        // 初始化每个主机的倒计时和加载状态
        const initialCountdowns: Record<string, number> = {};
        const initialLoadingStates: Record<string, boolean> = {};
        hostsData.forEach(host => {
          initialCountdowns[host.name] = REFRESH_INTERVAL;
          initialLoadingStates[host.name] = true;
        });
        setCountdowns(initialCountdowns);
        setLoadingHosts(initialLoadingStates);

        // 获取本地存储的唤醒设置
        const settings = pcStatusApi.getKeepAwakeSettings();
        
        // 逐个获取主机数据
        hostsData.forEach(host => {
          setRefreshingHosts(prev => ({ ...prev, [host.name]: true }));
          fetchHostData(host.name, settings[host.name]);
        });
      } catch (err) {
        const error = err as AxiosError<APIError>;
        console.error('获取主机列表失败:', error);
        message.error(`获取主机列表失败: ${error.response?.data?.error || error.message}`);
      }
    };

    loadHosts();
  }, []);

  // 每个主机的自动刷新倒计时
  useEffect(() => {
    const timer = setInterval(() => {
      setCountdowns(prev => {
        const newCountdowns = { ...prev };
        let needsUpdate = false;

        hosts.forEach(host => {
          if (newCountdowns[host.name] > 0) {
            newCountdowns[host.name]--;
            if (newCountdowns[host.name] === 0) {
              // 倒计时结束，刷新该主机数据
              const settings = pcStatusApi.getKeepAwakeSettings();
              fetchHostData(host.name, settings[host.name]);
              newCountdowns[host.name] = REFRESH_INTERVAL;
            }
            needsUpdate = true;
          }
        });

        return needsUpdate ? newCountdowns : prev;
      });
    }, 1000);

    return () => clearInterval(timer);
  }, [hosts]);

  const handleKeepAwakeChange = async (hostName: string, checked: boolean) => {
    try {
      pcStatusApi.setLocalKeepAwake(hostName, checked);
      if (checked) {
        await fetchHostData(hostName, true);
      } else {
        setHostStatuses(prev => ({
          ...prev,
          [hostName]: { ...prev[hostName], keepAwake: false }
        }));
      }
    } catch (err) {
      const error = err as AxiosError<APIError>;
      pcStatusApi.setLocalKeepAwake(hostName, !checked);
      console.error('设置唤醒状态失败:', error);
      message.error(`设置唤醒状态失败: ${error.response?.data?.error || error.message}`);
    }
  };

  const handleRefresh = (hostName: string) => {
    const settings = pcStatusApi.getKeepAwakeSettings();
    setRefreshingHosts(prev => ({ ...prev, [hostName]: true }));
    fetchHostData(hostName, settings[hostName]);
  };

  const clientInfoColumns = [
    { 
      title: '地址',
      key: 'ipPort',
      render: (_: unknown, record: ClientInfo) => (
        <span>{record.ip}:{record.port}</span>
      )
    },
    { 
      title: '平台',
      key: 'platform',
      render: (_: unknown, record: ClientInfo) => {
        const { platform } = parseUserAgent(record.userAgent);
        return platform;
      }
    },
    { 
      title: '浏览器',
      key: 'browser',
      render: (_: unknown, record: ClientInfo) => {
        const { browser } = parseUserAgent(record.userAgent);
        return browser;
      }
    },
    {
      title: 'User Agent',
      dataIndex: 'userAgent',
      key: 'userAgent',
      render: (userAgent: string) => (
        <Tooltip title={userAgent}>
          <span style={{ 
            maxWidth: '200px', 
            overflow: 'hidden', 
            textOverflow: 'ellipsis', 
            whiteSpace: 'nowrap',
            display: 'inline-block'
          }}>
            {userAgent}
          </span>
        </Tooltip>
      )
    },
    { 
      title: '最后在线时间', 
      dataIndex: 'lastSeen', 
      key: 'lastSeen',
      render: (time: string) => formatDate(time)
    }
  ];

  const channelColumns = [
    { title: '服务端口', dataIndex: 'service_port', key: 'service_port' },
    { title: '目标主机', dataIndex: 'target_host', key: 'target_host' },
    { title: '目标端口', dataIndex: 'target_port', key: 'target_port' },
    { 
      title: '活跃连接数',
      dataIndex: 'active_count',
      key: 'active_count',
      render: (count: number) => count || 0
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'active' ? 'green' : 'red'}>
          {status === 'active' ? '活跃' : '非活跃'}
        </Tag>
      ),
    },
    {
      title: '最后活跃时间',
      dataIndex: 'last_active',
      key: 'last_active',
      render: (time?: string) => time ? formatDate(time) : '-'
    }
  ];

  const channelClientColumns = [
    { title: '客户端IP', dataIndex: 'ip', key: 'ip' },
    { 
      title: '客户端端口',
      dataIndex: 'ports',
      key: 'ports',
      render: (ports: string[]) => {
        if (ports.length <= 3) {
          return ports.join(', ');
        }
        return (
          <Tooltip title={ports.join(', ')}>
            <span>{ports.slice(0, 2).join(', ')}... ({ports.length}个)</span>
          </Tooltip>
        );
      }
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: () => (
        <Tag color="green">活跃</Tag>
      ),
    },
    {
      title: '最后活跃时间',
      dataIndex: 'last_active',
      key: 'last_active',
      render: (time: string) => formatDate(time)
    }
  ];

  // 渲染主机卡片
  const renderHostCard = (host: PCHostInfo) => {
    const status = hostStatuses[host.name];
    const clients = hostClients[host.name] || [];
    const channels = hostChannels[host.name] || [];
    const countdown = countdowns[host.name] || REFRESH_INTERVAL;

    return (
      <Card 
        key={host.name}
        title={`${host.name} (${host.ip})`}
        loading={loadingHosts[host.name]}
        style={{ marginBottom: '24px' }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          <Tag color={status?.isOnline ? 'green' : 'red'}>
            {status?.isOnline ? '在线' : '离线'}
          </Tag>
          <Button 
            icon={<SyncOutlined spin={refreshingHosts[host.name]} />} 
            onClick={() => handleRefresh(host.name)}
          >
            刷新
          </Button>
          <span>{countdown}秒后自动刷新</span>
          <span style={{ marginLeft: 'auto' }}>保持唤醒：</span>
          <Switch 
            checked={status?.keepAwake}
            onChange={(checked) => handleKeepAwakeChange(host.name, checked)}
          />
          {status?.lastWakeTime && (
            <span>最后唤醒: {formatTimeAgo(status.lastWakeTime)}</span>
          )}
        </div>

        <Collapse ghost style={{ marginTop: '16px' }}>
          <Panel header="网页唤醒客户端" key="clients">
            <Table 
              columns={clientInfoColumns}
              dataSource={clients}
              rowKey="id"
              pagination={false}
            />
          </Panel>
          <Panel header="转发唤醒客户端" key="channels">
            <Table 
              columns={channelColumns}
              dataSource={channels}
              rowKey="id"
              pagination={false}
              expandable={{
                defaultExpandAllRows: false,
                expandedRowRender: (record: ForwardChannel) => (
                  <Table
                    columns={channelClientColumns}
                    dataSource={record.clients || []}
                    rowKey="id"
                    pagination={false}
                  />
                ),
              }}
            />
          </Panel>
        </Collapse>
      </Card>
    );
  };

  // 对主机列表进行排序
  const sortedHosts = [...hosts].sort((a, b) => {
    const statusA = hostStatuses[a.name];
    const statusB = hostStatuses[b.name];
    
    // 如果状态不存在，认为是离线
    const isOnlineA = statusA?.isOnline ?? false;
    const isOnlineB = statusB?.isOnline ?? false;

    // 在线的排在前面
    if (isOnlineA && !isOnlineB) return -1;
    if (!isOnlineA && isOnlineB) return 1;
    
    // 如果在线状态相同，按名称排序
    return a.name.localeCompare(b.name);
  });

  return (
    <div style={{ padding: '24px' }}>
      <Title level={2}>远程PC控制面板</Title>
      {sortedHosts.map(renderHostCard)}
    </div>
  );
};

export default RemoteControl; 