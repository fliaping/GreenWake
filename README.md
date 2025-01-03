# 远程PC唤醒与端口转发工具

一个基于 Web 的远程 PC 唤醒和端口转发管理工具，支持多主机管理、自动唤醒和端口转发， 尤其是转发前唤醒的功能，可以无痛使用休眠主机的服务， 例如通过一条ssh命令就能将远程主机唤醒并连接。

![页面截图](docs/screenshot.png)

## 功能特点

- 🖥️ 多主机管理：支持管理多台远程主机
- 🔄 自动唤醒：通过 WOL (Wake-on-LAN) 实现远程唤醒
- 🚀 端口转发：支持多端口转发配置，转发时唤醒
- 🔄 自动重试：主机唤醒失败时自动重试
- 📊 实时监控：显示主机状态、客户端连接信息
- 🌐 Web 界面：友好的 Web 管理界面

## 配置文件说明

```yaml
log:
  level: "debug"  # 日志级别：debug, info, warn, error

http:
  port: "8055"    # Web服务端口
  user: "admin"   # 管理员用户名
  password: "123456" # 管理员密码

hosts:  # 主机配置
  - name: "home-pc"        # 主机名称
    ip: "192.168.1.100"    # 主机IP
    mac: "XX:XX:XX:XX:XX:XX" # 主机MAC地址
    monitor_port: 3389     # 监控端口
    wake_timeout: 5        # 唤醒超时时间(秒)，默认10秒
    retry_count: 4         # 唤醒重试次数，默认1次

forwards:  # 端口转发配置
  - service_port: 13322    # 服务端监听端口
    target_host: "home-pc" # 目标主机名称
    target_port: 22022     # 目标主机端口
```

## 使用指南

### Docker 方式启动

1. 准备配置文件

```bash
vim ./config.yml  # 编辑配置文件
```

2. 运行容器

```bash
docker run -d \
  --name wol \
  --restart unless-stopped \
  -p 8055:8055 \
  -v ./config.yml:/app/config.yml \
  xuping/my-wol:latest
```

### 本地编译启动

1. 克隆代码

```bash
git clone https://github.com/fliaping/my-wol.git
cd wol
```

2. 编译运行

```bash
# 编译前端
cd web
npm install
npm run build

# 编译后端
cd ..
# 安装依赖
task install-deps

# 直接运行
go run cmd/server/main.go -config config.yml

# 或者构建&运行
go build -o wol ./cmd/server
./wol -config config.yml

```

## 开发指南

### 项目结构

```
.
├── cmd/
│   └── server/          # 程序入口
├── internal/
│   ├── api/            # HTTP API 处理
│   ├── config/         # 配置管理
│   ├── model/          # 数据模型
│   └── service/        # 业务逻辑
├── web/                # 前端代码
│   ├── src/
│   └── package.json
└── config.yml         # 配置文件
```

### 前端开发

1. 开发环境启动

```bash
cd web
npm install
npm run dev
```

2. 构建

```bash
npm run build
```

主要技术栈：

- React
- TypeScript
- Ant Design
- Vite

### 后端开发

1. 开发环境启动

```bash
go run ./cmd/server -config config.yml
```

2. 构建

```bash
go build -o wol ./cmd/server
```

主要功能模块：

- `api`: HTTP API 路由和处理
- `service/pc.go`: 主机管理和唤醒
- `service/forward.go`: 端口转发
- `config`: 配置文件处理

主要技术栈：

- Go
- Gin
- Wake-on-LAN
- TCP 端口转发

### API 接口

- `GET /api/pc/hosts`: 获取主机列表
- `GET /api/pc/:hostName/status`: 获取主机状态
- `GET /api/pc/:hostName/client_info`: 获取客户端信息
- `GET /api/pc/:hostName/forward_channels`: 获取转发通道信息

### Docker构建

多平台镜像构建

```shell
# 创建构建环境
docker buildx create --name mybuilder --driver docker-container --bootstrap
# 使用新创建的构建环境
docker buildx use mybuilder
# 构建并推送到仓库
docker buildx build --platform linux/amd64,linux/arm64 -t xuping/my-wol:latest --push .
```

### 开发注意事项

1. 前端开发时可以使用 mock 数据进行测试
2. 修改配置后需要重启服务
3. 开发时建议使用 debug 日志级别
4. 主机唤醒和端口转发功能需要在同一网段测试
