import { http, HttpResponse } from 'msw';

type PathParams = {
  hostName: string;
}

export const handlers = [
  // 主机列表接口
  http.get('/api/pc/hosts', () => {
    return HttpResponse.json({
      success: true,
      data: [
        {
          name: 'home-pc',
          ip: '192.168.1.100',
          mac: 'AA:BB:CC:DD:EE:FF',
          monitorPort: 3389
        },
        {
          name: 'office-pc',
          ip: '192.168.2.100',
          mac: '11:22:33:44:55:66',
          monitorPort: 22
        }
      ]
    });
  }),

  // 主机状态查询接口
  http.get<PathParams>('/api/pc/:hostName/status', ({ params, request }) => {
    const url = new URL(request.url);
    const keepAwake = url.searchParams.get('keepAwake') === 'true';

    return HttpResponse.json({
      success: true,
      data: {
        name: params.hostName,
        isOnline: true,
        keepAwake: keepAwake,
        lastUpdate: new Date().toISOString(),
        lastWakeTime: keepAwake ? new Date().toISOString() : undefined
      }
    });
  }),

  // 主机客户端信息接口
  http.get('/api/pc/:hostName/client_info', ({ }) => {
    return HttpResponse.json([
      {
        id: '1',
        userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)',
        ip: '192.168.1.10',
        port: '52100',
        lastSeen: new Date().toISOString()
      }
    ]);
  }),

  // 主机转发通道接口
  http.get('/api/pc/:hostName/forward_channels', ({ params }) => {
    const { hostName } = params;
    return HttpResponse.json([
      {
        id: '1',
        servicePort: hostName === 'home-pc' ? 13389 : 10022,
        targetPort: hostName === 'home-pc' ? 3389 : 22,
        status: 'active',
        lastActive: new Date().toISOString()
      }
    ]);
  })
]; 