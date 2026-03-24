# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在此代码库中工作提供指导。

## 项目概述

FreeKiosk Hub 是用于管理和监控 kiosk 设备集群的中央服务器，使用 Tailscale 实现安全的零配置网络，允许你从任何地方管理设备。

## 开发命令

```bash
# 安装/更新依赖
make deps

# 生成 UI 组件 (templ)
make generate

# 构建
make build

# 运行
make run

# 清理 (删除二进制文件和数据库)
make clean
```

### 手动命令

```bash
# 下载依赖
go mod tidy
go mod download

# 构建
go build -o bin/freekiosk-hub cmd/server/main.go

# 运行
go run cmd/server/main.go
```

## 架构概览

### 技术栈
- **Go**: 1.25.5+
- **Web 框架**: Echo v4
- **模板引擎**: Templ
- **数据库**: SQLite (sqlx + go-sqlite)
- **网络**: Tailscale API

### 项目结构

```
cmd/server/          # 主入口
internal/
  ├── api/          # HTTP 处理器和路由
  ├── config/       # 环境变量和配置
  ├── databases/    # 数据库连接和模式
  ├── models/       # 核心数据结构
  ├── repositories/ # 数据访问层
  ├── services/     # 业务逻辑
  ├── network/      # 外部服务客户端 (Tailscale)
  └── clients/      # Kiosk 客户端通信
ui/                 # Templ UI 组件
```

### 核心组件

| 文件 | 用途 |
|------|------|
| `cmd/server/main.go` | 应用入口 |
| `internal/api/router.go` | Echo 路由设置 |
| `internal/clients/kiosk_client.go` | 与 FreeKiosk 设备通信 |
| `internal/clients/tailscale.go` | Tailscale API 客户端 |
| `internal/services/kiosk_service.go` | Kiosk 业务逻辑 |
| `internal/services/monitor.go` | 设备状态监控 |
| `internal/sse/hub.go` | Server-Sent Events 实时更新 |
| `internal/config/config.go` | 配置管理 |

## 配置

通过 `.env` 文件或环境变量配置：

```dotenv
SERVER_PORT=8081          # Web 界面端口
DB_PATH=freekiosk.db      # SQLite 数据库路径
TS_AUTHKEY=tskey-xxx      # Tailscale API 密钥 (必需)
KIOSK_PORT=8080           # Kiosk 设备 API 端口
KIOSK_API_KEY=your-key    # Kiosk 共享密钥
POLL_INTERVAL=30s         # 状态轮询间隔
RETENTION_DAYS=31         # 历史数据保留天数
MAX_WORKERS=5             # 并发工作线程数
```

### 环境变量

| 变量 | 描述 | 默认值 |
|------|------|--------|
| `SERVER_PORT` | Web 界面端口 | 8081 |
| `DB_PATH` | SQLite 数据库路径 | freekiosk.db |
| `TS_AUTHKEY` | Tailscale API 密钥 | 必需 |
| `KIOSK_PORT` | Kiosk API 端口 | 8080 |
| `KIOSK_API_KEY` | Kiosk 共享密钥 | - |
| `POLL_INTERVAL` | 轮询间隔 | 30s |
| `RETENTION_DAYS` | 数据保留天数 | 31 |
| `MAX_WORKERS` | 并发工作线程 | 5 |

## Git Hooks 工作流

项目已配置 git hooks 来规范 commit message 格式：

```bash
# 配置 hooks（首次 clone 后运行）
bash bin/setup-hooks.sh

# 功能完成后运行此脚本（自动生成文档 + 更新 changelog + 版本递增）
bash bin/post-feature-hub.sh

# 或者手动提交时，遵循以下格式：
git commit -m "type(scope): description"
```

### Commit Message 格式

```
type(scope): description
```

**type 可选值：**
| type | 含义 |
|------|------|
| feat | 新功能 |
| fix | Bug 修复 |
| docs | 文档更新 |
| ui | UI/样式变更 |
| api | API 变更 |
| db | 数据库变更 |
| refactor | 重构 |
| build | 依赖/构建 |
| ci | CI 配置 |
| test | 测试相关 |

**示例：**
```bash
git commit -m "feat(api): 添加设备信息上报接口"
git commit -m "fix(hub): 修复SELECT * 导致sqlx扫描失败"
git commit -m "ui(dashboard): 优化仪表板布局"
```

**规则：**
- description 使用中文，句末不加句号
- 长度不超过 72 字符
- scope 可选（如 android、hub、api、ui）

- **Go**: 编程语言
- **Make**: 用于常见任务
- **Templ**: UI 模板生成器

安装 Templ:
```sh
go install github.com/a-h/templ/cmd/templ@latest
```

## 与客户端的关系

本项目是 **服务器端** 组件，需要与客户端应用配合使用：

- **服务器**: freekiosk-hub (本仓库)
- **客户端**: [FreeKiosk](https://github.com/RushB-fr/freekiosk/) (安装在 Android 平板上)
