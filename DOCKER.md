# FreeKiosk Hub Docker 部署指南

## 快速开始

### 1. 克隆仓库

```bash
cd freekiosk-hub
```

### 2. 配置环境变量

复制示例配置文件并填写你的设置：

```bash
cp .env.example .env
```

编辑 `.env` 文件：

```bash
# .env 文件内容示例
KIOSK_API_KEY=your-secret-api-key-here
TS_AUTHKEY=tskey-auth-your-key-here
BASE_URL=http://your-server-ip:8081
```

**重要**: `TS_AUTHKEY` 是必需的，需要从 Tailscale 控制台获取：
1. 访问 https://login.tailscale.com/admin/settings/keys
2. 创建一个新的认证密钥
3. 复制密钥到 `.env` 文件

### 3. 构建并启动

```bash
# 使用 docker-compose 构建并启动
docker compose up --build -d
```

### 4. 访问 Web 界面

打开浏览器访问：`http://localhost:8081`

## 命令参考

### 启动服务

```bash
# 后台启动
docker compose up -d

# 前台启动（查看日志）
docker compose up
```

### 停止服务

```bash
# 正常停止
docker compose down

# 停止并删除数据卷（谨慎使用！）
docker compose down -v
```

### 查看日志

```bash
# 查看所有日志
docker compose logs

# 实时查看日志
docker compose logs -f

# 查看特定服务日志
docker compose logs freekiosk-hub
```

### 重启服务

```bash
docker compose restart
```

### 重新构建

```bash
docker compose build --no-cache
```

## 数据持久化

数据存储在两个 Docker 卷中：

- `freekiosk-hub-data`: SQLite 数据库
- `freekiosk-hub-media`: 媒体文件

### 备份数据

```bash
# 备份数据库
docker run --rm \
  -v freekiosk-hub-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/freekiosk-data-backup.tar.gz -C /data .

# 备份媒体文件
docker run --rm \
  -v freekiosk-hub-media:/media \
  -v $(pwd):/backup \
  alpine tar czf /backup/freekiosk-media-backup.tar.gz -C /media .
```

### 恢复数据

```bash
# 恢复数据库
docker run --rm \
  -v freekiosk-hub-data:/data \
  -v $(pwd):/backup \
  alpine tar xzf /backup/freekiosk-data-backup.tar.gz -C /data

# 恢复媒体文件
docker run --rm \
  -v freekiosk-hub-media:/media \
  -v $(pwd):/backup \
  alpine tar xzf /backup/freekiosk-media-backup.tar.gz -C /media
```

## 环境变量

| 变量 | 描述 | 默认值 | 必需 |
|------|------|--------|------|
| `SERVER_PORT` | Web 界面端口 | 8081 | 否 |
| `LOG_LEVEL` | 日志级别 (DEBUG/INFO/WARN/ERROR) | INFO | 否 |
| `KIOSK_PORT` | Kiosk API 端口 | 8080 | 否 |
| `KIOSK_API_KEY` | Kiosk 共享密钥 | - | **是** |
| `TS_AUTHKEY` | Tailscale 认证密钥 | - | **是** |
| `POLL_INTERVAL` | 轮询间隔 (如 30s, 1m) | 30s | 否 |
| `RETENTION_DAYS` | 数据保留天数 | 31 | 否 |
| `MAX_WORKERS` | 并发工作线程数 | 5 | 否 |
| `BASE_URL` | 服务器基础 URL | http://localhost:8081 | 否 |

## 端口说明

- **8081**: Web 管理界面
- **8080**: Kiosk 设备 API（内部使用）

## Tailscale 配置

FreeKiosk Hub 使用 Tailscale 与 kiosk 设备通信。

### 获取 Tailscale 密钥

1. 登录 [Tailscale 控制台](https://login.tailscale.com/admin/settings/keys)
2. 点击 "Generate auth key"
3. 复制密钥到 `.env` 文件的 `TS_AUTHKEY` 变量

### 在 Tailnet 中添加设备

在 kiosk 设备上安装 Tailscale：

```bash
# Android 设备安装 Tailscale
# 从 Google Play Store 下载 Tailscale 应用
```

## 故障排除

### 容器无法启动

```bash
# 查看详细日志
docker compose logs freekiosk-hub
```

### 数据库损坏

```bash
# 停止服务
docker compose down

# 删除数据卷（数据将丢失！）
docker volume rm freekiosk-hub-data

# 重新启动（创建新数据库）
docker compose up -d
```

### Tailscale 连接失败

确保：
1. `TS_AUTHKEY` 正确且未过期
2. 你的 Tailnet 未满
3. 防火墙允许出站连接

## 更新

```bash
# 拉取最新代码
git pull

# 重新构建并重启
docker compose up -d --build
```

## 安全建议

1. **修改默认密钥**: 确保 `KIOSK_API_KEY` 使用强密码
2. **保护 Tailscale 密钥**: 不要将 `.env` 文件提交到版本控制
3. **使用 HTTPS**: 在生产环境中，考虑在 Docker 前使用反向代理（如 Nginx）配置 HTTPS
4. **定期备份**: 定期备份数据卷
