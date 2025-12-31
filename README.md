# Go Bot

这是原 Python 机器人的 Go 语言重写版本。

## 功能
- **AI 对话**: 集成 Google Gemini (支持多 Key 轮询、模型降级)。
- **OpenWrt 管理**: 通过 SSH 获取系统状态。
- **权限管理**: 简单的管理员验证。

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
- 目前仅实现了核心 AI 和 SSH 查询功能，更多插件正在迁移中。
