# Go Bot

这是原 Python 机器人的 Go 语言重写版本。

## 功能
- **AI 对话**: 集成 Google Gemini (支持多 Key 轮询、模型降级、上下文记忆)。
- **OpenWrt 管理**:
  - 系统状态监控 (CPU/内存/负载)
  - 设备列表扫描 (DHCP/ARP)
  - AdGuard Home 管理 (查看统计/拦截开关)
  - 网络工具箱 (Ping/Trace/Nslookup)
- **OpenClash 控制**: 状态查看、模式切换、日志分析。
- **实用工具**:
  - 贴纸/图片格式转换
  - 临时邮箱生成
- **权限管理**: 基于 ID 的白名单验证。

## 配置说明
项目使用 `.env` 文件进行配置。请复制演示文件并修改为实际值：

1. 复制 `.env.example` 为 `.env`：
   ```bash
   cp .env.example .env
   ```
2. 编辑 `.env` 文件，填入 Bot Token、API Key 等信息。
   > **注意**: `GEMINI_API_KEY` 支持配置多个 Key（用逗号分隔）以实现自动轮询和负载均衡。

## 目录结构
```
.
├── config/      # 配置加载
├── pkg/
│   ├── ai/      # Gemini 客户端
│   ├── bot/     # Telegram Bot 核心逻辑
│   ├── openwrt/ # OpenWrt/SSH 客户端
│   ├── openclash/ # OpenClash 客户端
│   └── utils/   # 工具函数
├── main.go      # 入口文件
├── go.mod
├── Dockerfile
└── docker-compose.yml
```

## 运行方式

### 1. Docker 运行 (推荐)
支持使用预编译镜像或 Docker Compose 一键部署。

#### 方式 A：使用 Docker Compose (包含自动更新)
```yaml
# docker-compose.yml
services:
  bot:
    image: yingxiaomo/homeops:latest
    container_name: homeops_bot
    restart: unless-stopped
    env_file: .env
    network_mode: host
    volumes:
      - ./data:/app/data

  # 自动更新服务 (可选)
  watchtower:
    image: containrrr/watchtower
    container_name: watchtower
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    command: --interval 300 --cleanup
```

启动命令：
```bash
docker-compose up -d
```

#### 方式 B：手动构建
如果您想运行本地修改后的版本：

```bash
docker-compose up -d --build
```

### 2. 本地运行
需要安装 Go 1.22+。

```bash
# Windows
go mod tidy
go build -o bot.exe
./bot.exe

# Linux/Mac
go mod tidy
go run main.go
```

## 注意事项
- 确保 `.env` 文件中有 `TG_BOT_TOKEN`, `GEMINI_API_KEY`, `OPENWRT_HOST` 等配置。
