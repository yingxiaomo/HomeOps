# 🏠 HomeOps - 家庭实验室全能管家

**HomeOps** 是一个基于 Python 和 Telegram Bot 的智能化家庭网络运维助手。它集成了 OpenWrt 系统管理、OpenClash 代理控制、网络诊断工具箱以及 **Gemini 3.0 AI 助手**，致力于为极客提供从故障排查到日常管理的“上帝视角”。

## ✨ 核心功能

### 🧠 1. 强力 AI 助手 (Gemini 3.0)
*   **沉浸式对话**：支持多轮上下文记忆，不仅是发指令，更能陪你聊技术。
*   **多模态识别**：发送图片，AI 帮你分析；发送 GitHub 链接，AI 帮你读代码。
*   **智能故障诊断**：一键抓取 OpenWrt 系统日志或 OpenClash 内核日志，AI 自动分析报错原因。
*   **究极降级链条**：内置模型自动轮询机制 (3.0 Pro -> 2.5 Pro -> 1.5 Pro -> 3.0 Flash...)，结合多 API Key 轮询，最大限度利用免费额度，永不掉线。

### 🔐 2. 安全权限系统
*   **严格鉴权**：核心功能（路由器管理、代理控制）仅限管理员（Admin ID）访问。
*   **动态授权**：管理员可通过 `/grant <用户ID> ai` 指令，安全地将 AI 等非敏感功能开放给家人或朋友。

### 🚀 3. OpenClash 深度集成
*   **状态监控**：实时查看节点延迟、流量消耗。
*   **动态调试**：无需重启，一键切换内核 `Info/Debug` 日志级别，配合 AI 分析快速定位翻墙故障。
*   **工具箱**：一键测速、清理 FakeIP、重载配置。

### 📟 4. OpenWrt 全能管理
*   **精准 IP 监测**：基于 `ubus` 原生指令，精准获取运营商分配的真实公网 IP（避开代理干扰），支持多接口自动探测。
*   **变动通知**：IP 变动秒级推送。
*   **设备管理**：查看内网在线设备。
*   **网络工具**：集成 Ping、Traceroute、Nslookup、Curl 等诊断工具。

### 🛠 5. 实用小工具
*   **临时邮箱**：集成 1secmail，一键生成临时邮箱收验证码。
*   **贴纸转换**：发送贴纸，自动转换为 PNG 图片。

---

## 🛠 部署指南

### 方式一：Docker 容器化部署 (推荐)

最稳定、最简单的部署方式，支持 Watchtower 自动更新。

**1. 创建目录与配置文件**
```bash
mkdir homeops && cd homeops
nano .env
```

**2. 配置环境变量 (.env)**
请参考下方配置模板，填入您的信息。

```env
# --- 🤖 机器人核心 ---
BOT_TOKEN=你的_Telegram_Bot_Token
ADMIN_ID=你的_Telegram_User_ID

# --- 🧠 Gemini AI (支持多Key轮询) ---
# 用逗号分隔多个 Key，额度耗尽自动切换
GEMINI_API_KEY=key1,key2,key3

# --- 🌐 网络配置 (可选) ---
# 自定义 API 反代 (推荐国内服务器使用)
# TG_BASE_URL=https://your-worker-domain.com/bot
# 或使用 HTTP 代理
# TG_PROXY=http://192.168.0.10:7890

# --- 📟 OpenWrt 管理 (SSH) ---
OPENWRT_HOST=192.168.1.1
OPENWRT_PORT=22
OPENWRT_USER=root
OPENWRT_PASS=你的_SSH_密码

# --- 🚀 OpenClash 管理 (API) ---
OPENCLASH_API_URL=http://192.168.1.1:9090
OPENCLASH_API_SECRET=你的_API_Secret
```

**3. 创建 docker-compose.yml**

```yaml
services:
  bot:
    image: yingxiaomo/homeops:latest 
    container_name: homeops_bot
    restart: unless-stopped
    env_file: .env
    # 使用 host 模式以直接访问局域网设备 (OpenWrt)
    network_mode: host 
    volumes:
      - ./data:/app/data       # 持久化数据 (IP历史、权限配置)
      # - ./config:/app/config   # ⚠️ 注意：仅在使用 git clone 完整下载源码时才挂载此目录，否则会导致报错
      # - ./plugins:/app/plugins # ⚠️ 注意：仅在使用 git clone 完整下载源码时才挂载此目录

  # 自动更新服务 (每5分钟检查一次)
  watchtower:
    image: containrrr/watchtower
    container_name: watchtower
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    command: --interval 300 --cleanup
    environment:
      - TZ=Asia/Shanghai
```

**4. 启动服务**

```bash
docker-compose up -d
```

---

### 方式二：Python 源码运行 (开发调试)

```bash
# 1. 克隆代码
git clone https://github.com/yingxiaomo/HomeOps.git
cd HomeOps

# 2. 创建环境
python -m venv .venv
source .venv/bin/activate  # Linux/Mac
# .venv\Scripts\activate   # Windows

# 3. 安装依赖 (包含 AI 组件)
pip install -r requirements.txt

# 4. 配置 .env 并运行
cp .env.example .env
# 编辑 .env ...
python main.py
```

---

## 🎮 使用说明

### 常用指令
*   `/start` - 呼出主控台 (AI / OpenClash / OpenWrt / 工具)
*   `/ai` - 快速进入 AI 沉浸对话模式
*   `/id` - 获取当前用户 ID (用于授权)

### 管理员指令
*   `/grant <用户ID> ai` - 授权用户使用 AI 功能
*   `/revoke <用户ID> ai` - 撤销权限
*   `/users` - 查看已授权用户列表

## ⚠️ 注意事项
*   **OpenWrt SSH**: 请确保运行机器人的设备能通过 SSH 连接到路由器。
*   **OpenClash API**: 请确保在 OpenClash 插件设置中开启了 "允许局域网访问控制面板"，并配置了正确的端口和密钥。
*   **AI 额度**: 虽然内置了降级链，但建议至少配置 2-3 个 Google API Key 以保证高强度使用下的稳定性。

---
*Built with ❤️ by HomeOps Team*