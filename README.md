# mcstatus

一个用 Go 编写的高性能 Minecraft 服务器状态查询 RESTful API 服务，完全兼容 [api.mcstatus.io](https://api.mcstatus.io) 协议规范。基于官方底层库 [mcutil](https://github.com/mcstatus-io/mcutil) 开发，可作为官方 API 失效后的完美自建替代方案。

## ✨ 特性

- **协议兼容**：100% 兼容 `api.mcstatus.io` v2 RESTful 协议接口与响应数据结构。
- **双端支持**：完整支持 Java 版（Modern 1.7+、Legacy 及 GS4 Query 插件/软件探测）与基岩版（Bedrock）。
- **零配置开箱即用**：缺省所有配置项即可直接启动运行。
- **内存缓存 & Cloudflare CDN 友好**：内置带并发合并（SingleFlight）的高性能内存 TTL 缓存，精准输出 `Cache-Control`、`CDN-Cache-Control`、`Cloudflare-CDN-Cache-Control`、`ETag` 与 `X-Cache-Hit` 标头，支持 304 Not Modified。
- **真实 IP 与彩色日志**：基于 `TRUSTED_PROXIES` 受信代理自动安全提取 `CF-Connecting-IP`、`X-Real-IP`、`X-Forwarded-For` 真实访客 IP，默认覆盖所有私网网段防伪造，终端基于 `tint` 提供现代化着色日志。
- **现代化构建**：使用 [ko](https://ko.build/) 构建轻量多架构容器镜像，底层使用 `alpine:latest`，推送到 `main` 分支自动发布至 GHCR。
- **AGPLv3 开源协议**。

---

## 🚀 快速启动

### 方式一：Docker Compose（推荐）

创建 `docker-compose.yml`：

```yaml
services:
  mcstatus:
    image: ghcr.io/ai-trash/mcstatus:latest
    container_name: mcstatus
    restart: unless-stopped
    ports:
      - "3001:3001"
    environment:
      - PORT=3001
      - CACHE_TTL=60s
      - DEFAULT_TIMEOUT=5s
```

启动服务：

```bash
docker compose up -d
```

### 方式二：Docker Run

```bash
docker run -d \
  --name mcstatus \
  --restart unless-stopped \
  -p 3001:3001 \
  ghcr.io/ai-trash/mcstatus:latest
```

### 方式三：源码运行

```bash
go run ./cmd/server
```

---

## 📡 API 接口一览

所有接口均位于 `/v2` 前缀下（同时兼容无 `/v2` 前缀的简写路径）：

| 接口 | 方法 | 说明 | 常用参数 |
| :--- | :--- | :--- | :--- |
| `/v2/status/java/:address` | `GET` | 查询 Java 版服务器状态 | `query=true`（默认 true）、`timeout=5.0` |
| `/v2/status/bedrock/:address` | `GET` | 查询基岩版服务器状态 | `timeout=5.0` |
| `/v2/icon/:address` | `GET` | 获取 Java 版服务器图标（PNG） | `timeout=5.0`（缺失或离线回退默认图标） |
| `/v2/icon` | `GET` | 获取默认服务器图标（PNG） | — |
| `/v2/widget/java/:address` | `GET` | 生成 Java 版卡片挂件（860×240 PNG） | `dark=true`、`rounded=true`、`transparent=false` |
| `/v2/widget/bedrock/:address` | `GET` | 生成基岩版卡片挂件（860×240 PNG） | `dark=true`、`rounded=true`、`transparent=false` |
| `/v2/vote` | `POST` | 发送 Votifier 投票 | 支持 Votifier 1 & 2 协议参数 |
| `/health` | `GET` | 健康检查探针 | — |

---

## ⚙️ 环境变量配置（可选）

本项目设计为**零配置启动**，以下环境变量均有合理默认值：

| 变量名 | 默认值 | 说明 |
| :--- | :--- | :--- |
| `HOST` | `0.0.0.0` | HTTP 监听地址 |
| `PORT` | `3001` | HTTP 监听端口 |
| `CACHE_TTL` | `60s` | 内存缓存及 CDN 缓存有效期 |
| `DEFAULT_TIMEOUT` | `5s` | 服务器状态查询默认超时时间 |
| `MAX_TIMEOUT` | `15s` | 服务器状态查询最大超时限制 |
| `MOJANG_BLOCKED_REFRESH` | `1h` | Mojang 封禁列表定时刷新间隔 |
| `TRUSTED_PROXIES` | 默认所有私网网段 | 受信代理 CIDR 列表（逗号分隔，如 `127.0.0.1/32,10.0.0.0/8` 或 `*` 全信任）。受信对端方可穿透读取 `CF-Connecting-IP`、`X-Real-IP` 与 `X-Forwarded-For` 客户端真实 IP，防止伪造。缺省默认覆盖所有 RFC 1918、CGNAT、环回及 IPv6 本地私网地址段。 |
---

## 📄 开源许可证

本项目基于 [GNU Affero General Public License v3.0 (AGPLv3)](./LICENSE) 协议开源。
