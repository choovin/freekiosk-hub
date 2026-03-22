# FreeKiosk 研学版 (Field Trip Edition) — 开发总结

> 本文档记录研学版功能从需求到实现的完整工程过程，包含架构决策、代码组织、以及踩坑记录。

---

## 1. 项目背景与目标

研学版面向中小学研学旅行场景，要求：

1. **快速绑定**：教师批量注册平板，学生扫码 ≤ 5 秒完成绑定
2. **应用锁定**：远程配置平板可用应用，非白名单 App 被拦截
3. **活动广播**：教师发送通知，所有平板即时全屏展示
4. **OTA 升级**：远程推送 APK，平板自动下载安装
5. **位置监控**：实时地图显示所有平板 GPS 位置

---

## 2. 架构决策

### 2.1 为什么用 MQTT 而不是纯 HTTP 轮询？

**决策：MQTT 主推 + HTTP 降级**

- MQTT 优势：广播消息延迟 < 100ms，无需平板主动请求
- MQTT 劣势：需要 MQTT Broker（EMQX），增加运维复杂度
- HTTP 降级：MQTT 不可用时，平板每 30 秒轮询 `/commands` 获取广播
- **结论**：MQTT 是体验最优解，HTTP 降级保证离线可用性

### 2.2 为什么用 SQLite 而不是 PostgreSQL？

**决策：SQLite（开发版），支持 PostgreSQL（生产版可扩展）

- 研学版场景：单 Hub，设备数量 ≤ 1000 台
- SQLite 足够，且运维简单（零配置）
- 已有架构支持 PostgreSQL（Phase 2 企业版已有迁移脚本）

### 2.3 为什么 QR 绑定用 group_key + api_key 两套密钥？

**决策：分离绑定认证和 API 认证**

| 密钥 | 用途 | 使用场景 |
|------|------|---------|
| `group_key` | 验证分组身份 | 平板绑定时确认加入了正确的分组 |
| `api_key` | 所有 API 请求认证 | GPS 上报、命令轮询等日常操作 |

- `group_key`：分组级别，所有设备共享同一密钥
- `api_key`：设备级别，每台设备独立密钥
- 安全：`api_key_hash` 存储 SHA-256 哈希值，不存明文

### 2.4 为什么 GPS 上报用 HTTP POST 而不是 MQTT？

**决策：HTTP POST**

- GPS 数据量大（每 30 秒一条），MQTT 适合小消息广播
- GPS 数据需要持久化存储（gps_logs 表），HTTP POST 直写 DB 更简单
- MQTT 专用于广播类消息（broadcast，命令下发）

---

## 3. 核心数据流

### 3.1 绑定流程（5 秒目标）

```
Teacher Hub UI
  │
  ├─ 创建分组 → group_key 自动生成
  ├─ 创建设备 → api_key 自动生成
  └─ 显示设备 QR 码（包含 device_id, group_key, api_key, hub_url）

Student Tablet
  │
  ├─ 打开 QR 扫描 → CameraX + ZXing 识别
  ├─ 解析 QR JSON → HubConfigModule.bindWithQrPayload()
  ├─ POST /api/v2/fieldtrip/devices/bind
  ├─ Hub 验证 api_key_hash → 写入 DB: status='active'
  ├─ 保存到 SharedPreferences
  ├─ 启动 GPS 上报（30s 间隔）
  └─ 启动 MQTT 订阅 fieldtrip/{group_id}/broadcast
```

### 3.2 广播流程

```
Teacher Hub UI → POST /api/v2/fieldtrip/broadcast
  │
  ├─ BroadcastService.SendToGroup()
  │    ├─ 写入 broadcasts 表
  │    └─ MQTTService.Publish(fieldtrip/{group_id}/broadcast, message)
  │         │
  │         └─ EMQX Broker 转发
  │
  └─ Tablet MQTT Client
       ├─ 收到消息
       ├─ KioskMqttClient.handleBroadcastMessage()
       └─ BroadcastOverlayActivity（全屏展示 + 提示音）
```

---

## 4. 目录结构

### 4.1 Hub (Go)

```
freekiosk-hub/
├── cmd/server/main.go                    # 入口
├── internal/
│   ├── api/
│   │   ├── router.go                   # Echo 路由（setupRoutes）
│   │   ├── fieldtrip_handler.go        # FieldTripHandler (15个端点)
│   │   ├── fieldtrip_ui.go           # Web UI Handler
│   │   └── ota_handler.go            # OTA 上传/列表
│   ├── models/
│   │   └── fieldtrip.go              # 模型定义
│   ├── repositories/
│   │   └── fieldtrip_repo.go        # SQLite CRUD
│   ├── services/
│   │   └── broadcast_service.go     # 广播 + MQTT 发布
│   └── mqtt/
│       ├── client.go                  # paho/auto-paho MQTT 客户端
│       └── config.go                 # 配置结构
├── ui/
│   └── fieldtrip.templ               # Templ UI 组件
└── docs/field-trip/
    ├── README.md                     # 用户文档
    └── ENGINEERING.md               # 本文档
```

### 4.2 Android (Kotlin)

```
freekiosk/
├── android/app/src/main/java/com/freekiosk/
│   ├── QrScannerModule.kt           # CameraX + ZXing
│   ├── QrScannerPackage.kt          # ReactPackage
│   ├── HubConfigModule.kt          # 配置 + GPS + 轮询
│   ├── BroadcastOverlayActivity.kt  # 广播通知 UI
│   ├── AppWhitelistManager.kt       # 白名单检查
│   ├── mqtt/
│   │   ├── KioskMqttClient.kt     # MQTT 订阅广播
│   │   └── MqttModule.kt          # RN 桥接
│   ├── AppLauncherModule.kt         # 集成白名单检查
│   └── api/KioskHttpServer.kt     # 平板 HTTP 服务端
└── app/build.gradle                # CameraX + ZXing + PlayServicesLocation
```

---

## 5. API 设计

### 5.1 RESTful 原则

所有端点遵循 REST 风格：
- `POST /groups` — 创建
- `GET /groups` — 列表
- `DELETE /groups/:id` — 删除

### 5.2 认证方案

```
Header: X-Api-Key: <device_api_key>
```

- 服务器在 middleware 或 handler 中读取 `X-Api-Key`
- 计算 SHA-256 与数据库 `api_key_hash` 比对
- 注意：**`X-Api-Key` 大小写敏感**，曾是 bug（修复自 e556b28）

### 5.3 响应规范

| 状态码 | 含义 |
|--------|------|
| 200 | 成功（有响应体） |
| 201 | 创建成功 |
| 204 | 成功（无响应体，如 GPS 上报） |
| 400 | 请求参数错误 |
| 401 | API Key 认证失败 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 6. 数据库设计

### 6.1 核心表

- `fieldtrip_groups`：分组（id, name, group_key）
- `fieldtrip_devices`：设备（id, name, group_id, api_key_hash, hub_url, status）
- `gps_logs`：GPS 日志（device_id, lat, lng, accuracy, timestamp）
- `broadcasts`：广播记录（id, group_id, message, delivered_count）
- `pending_commands`：待处理命令（device_id, command_type, payload, status）

### 6.2 索引策略

```sql
CREATE INDEX idx_gps_device ON gps_logs(device_id);
CREATE INDEX idx_gps_timestamp ON gps_logs(timestamp);
CREATE INDEX idx_pc_device ON pending_commands(device_id, status);
```

---

## 7. 踩坑记录

### 7.1 MQTT 初始连接阻塞启动

**问题：** `mqttClient.Connect(ctx)` 的 `AwaitConnection()` 会阻塞 10 秒（ConnectTimeout），导致 HTTP 服务器在这段时间内无法启动。

**原因：** 当没有 MQTT Broker 时，`AwaitConnection()` 会等待超时才返回失败。

**修复：** 使用独立 5 秒 context 调用 `AwaitConnection()`，主启动流程立即继续，AutoReconnect 在后台重连。

```go
// 修复前
cm, err := mqtt.NewConnection(ctx, clientConfig)
if err != nil { return err }
if err := cm.AwaitConnection(ctx); err != nil { return err }

// 修复后
connectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
cm, err := mqtt.NewConnection(ctx, clientConfig)
if err != nil { return err }
c.connection = cm // 先设置连接对象
if err := cm.AwaitConnection(connectCtx); err != nil {
    log.Printf("[MQTT] 初始连接失败，将在后台重试")
    return nil // 不阻塞
}
```

### 7.2 BroadcastService 未连接到 Handler

**问题：** `SendBroadcast` 只写 DB，从未调用 MQTT 发布。

**发现：** QA 测试时广播消息发送成功，但平板通过 MQTT 收不到。

**修复：** 在 `NewFieldTripHandler` 时传入 `BroadcastService`，调用 `BcastSvc.SendToGroup()`。

### 7.3 X-Api-Key 大小写不一致

**问题：** 服务器读取 `X-API-Key`（全大写），Android 客户端发送 `X-Api-Key`（混合大小写）。

**修复：** 统一为 `X-Api-Key`，Android 端和 Go 服务器端保持一致。

### 7.4 HubConfigModule 的 Context 引用

**问题：** `HubConfigModule` 中使用 `ContextCompat.checkSelfPermission()` 报错。

**原因：** Kotlin 混淆了 `android.content.ContextCompat` 和 `androidx.core.content.ContextCompat`。

**修复：** 使用导入的 `androidx.core.content.ContextCompat`，不使用 FQN 引用。

### 7.5 Echo v4 的 `c.SaveFile` 不存在

**问题：** OTA handler 使用 `c.SaveFile(file, dstPath)` 但 Echo v4 没有这个方法。

**修复：** 使用 `os.Create()` + `io.Copy()` 手动处理文件上传。

### 7.6 FieldTripUIHandler 返回值错误

**问题：** `CreateGroup` 方法返回 `(c echo.Context, error)` 但 Go 不支持多个返回值作为 Echo Handler。

**修复：** 统一 Handler 签名为 `func(c echo.Context) error`。

### 7.7 MQTT IsConnected 只检查指针非 nil

**问题：** `IsConnected()` 返回 `c.connection != nil`，但 `NewConnection()` 后即使连接失败也非 nil。

**修复：** 添加 `connected` 字段，通过 `OnConnectionUp`/`OnConnectionDown` 回调更新状态。

---

## 8. 测试策略

### 8.1 API 测试

使用 curl 手动测试所有端点：

```bash
# 基础测试
curl http://localhost:8081/health
curl http://localhost:8081/health/mqtt

# 分组 CRUD
curl -X POST http://localhost:8081/api/v2/fieldtrip/groups \
  -H "Content-Type: application/json" \
  -d '{"name":"测试组"}'
curl http://localhost:8081/api/v2/fieldtrip/groups

# 设备绑定
curl -X POST http://localhost:8081/api/v2/fieldtrip/devices/bind \
  -H "X-Api-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"device_id":"...","group_key":"...","api_key":"...","hub_url":"..."}'

# GPS 测试
curl -X POST "http://localhost:8081/api/v2/fieldtrip/devices/$ID/location" \
  -H "X-Api-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"lat":31.23,"lng":121.47,"accuracy":10,"timestamp":1774150700}'
```

### 8.2 健康检查覆盖

| 端点 | 验证内容 |
|------|---------|
| `/health` | 返回 `{"status":"ok","database":"connected"}` |
| `/health/mqtt` | 实际连接状态（非仅指针非 nil） |

---

## 9. 提交记录

### 9.1 Hub (freekiosk-hub)

| Commit | 描述 |
|--------|------|
| `871eb8c` | MQTT 5.0 客户端集成 |
| `9740a09` | MQTT 集成测试 |
| `0d3e08d` | MQTT 配置支持 |
| `5ee0fd9` | TLS 加密连接 |
| `afe89cd` | 修复 autopaho TLS 字段名 |
| `a1b5d4b` | MQTT 集成到主程序 |
| `7e01304` | Phase 2 企业认证 |
| `0edd2d7` | Phase 3 命令系统 + WebSocket |
| `585cde9` | Phase 4 安全策略 |
| `c2935e3` | Phase 5 多租户 |
| `a722ab5` | EMQX Docker 集成 |
| `e1bb576` | Phase 6 指标 + 审计日志 |
| `af0d5d9` | CA 证书加载 |
| `f26eec0` | Dockerfile 更新 |
| `614da54` | Field Trip 模型和仓库 |
| `20319df` | Field Trip API 处理器 + OTA |
| `9561f10` | Field Trip Web UI |
| `e556b28` | 修复 API 规范符合性 |
| `944f401` | 修复 MQTT 启动阻塞 |
| `cf2c5c0` | 修复 MQTT 健康检查准确性 |

### 9.2 Android (freekiosk)

| Commit | 描述 |
|--------|------|
| `a64679a` | HTTP 服务添加 Field Trip 路由 |
| `3df196d` | QR 扫描 + HubConfig 模块 |
| `0a5d65f` | MQTT 广播订阅 + 白名单 + Overlay |

---

## 10. 未来扩展

### 10.1 Phase 2 规划（见 TODOS.md）

1. **GPS 防抖**：50 米以内移动不重复上报，节省电量和 DB 写入
2. **嵌入式 MQTT Broker**：无外部 Broker 时 Hub 内置 moquette/paho server
3. **电子围栏**：定义地理围栏区域，设备离开时告警

### 10.2 已知的优化空间

| 项目 | 当前状态 | 理想状态 |
|------|---------|---------|
| PATCH 设备返回 204 | 无响应体 | 返回更新后的设备对象 |
| GPS 历史查询 | 返回固定数量 | 支持 limit/offset 分页 |
| OTA 签名验证 | 未来功能 | Ed25519 签名验证 |
| 白名单推送 | 仅推送到设备 | 支持批量设备同时推送 |

---

*本文档由 Claude Code 自动生成，记录于 2026-03-22*
