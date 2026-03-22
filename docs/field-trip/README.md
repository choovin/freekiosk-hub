# FreeKiosk 研学版 (Field Trip Edition) — 技术知识库

> 本文档详细记录 FreeKiosk 研学版的产品设计、系统架构、API 规范、数据库结构和部署运维指南。
>
> **版本：** 1.0.0
> **日期：** 2026-03-22
> **适用版本：** Hub ≥ 0.0.2，Android ≥ 支持 Field Trip Edition

---

## 目录

1. [产品概述](#1-产品概述)
2. [系统架构](#2-系统架构)
3. [Hub 服务器组件](#3-hub-服务器组件)
4. [Android 平板应用组件](#4-android-平板应用组件)
5. [API 接口规范](#5-api-接口规范)
6. [数据库结构](#6-数据库结构)
7. [功能模块详解](#7-功能模块详解)
8. [部署指南](#8-部署指南)
9. [运维指南](#9-运维指南)
10. [常见问题](#10-常见问题)

---

## 1. 产品概述

### 1.1 什么是研学版

研学版是 FreeKiosk 的一个专用 Edition，面向中小学研学旅行场景设计。老师通过 Hub 管理平台批量管理数十台 Android 平板，学生通过平板扫描 QR 码快速绑定到系统，系统支持：

- **快速绑定**：扫描 QR 码，5 秒内完成平板注册和配置
- **应用管理**：远程推送白名单应用，非白名单应用被锁定
- **活动广播**：教师发送广播消息，所有平板即时显示全屏通知
- **OTA 升级**：远程推送 APK 升级包，平板自动下载安装重启
- **地理位置监控**：实时查看所有平板的 GPS 位置

### 1.2 核心设计原则

```
┌─────────────────────────────────────────────────────┐
│                 研学版核心设计原则                      │
├─────────────────────────────────────────────────────┤
│ 1. 离线优先：MQTT 断线时自动降级 HTTP 轮询             │
│ 2. 快速绑定：QR 扫描 → 绑定 ≤ 5 秒                   │
│ 3. 零配置：平板扫码后自动获取 Hub URL、API Key        │
│ 4. 安全：API Key Hash 存储，MQTT TLS 加密           │
│ 5. 可观测：SSE 实时推送，Prometheus 指标              │
└─────────────────────────────────────────────────────┘
```

### 1.3 与标准版的区别

| 功能 | 标准版 (Kiosk) | 研学版 (Field Trip) |
|------|---------------|---------------------|
| 设备绑定 | ADB 配置 | QR 扫描绑定 |
| 应用控制 | 固定白名单 | 动态白名单管理 |
| 消息推送 | MQTT (需 Broker) | MQTT + HTTP 降级 |
| 地图视图 | 无 | GPS 实时位置 |
| 批量管理 | 分组管理 | 研学分组 + 批量命令 |
| OTA 推送 | 无 | APK 上传 + 推送 |

---

## 2. 系统架构

### 2.1 整体架构图

```
                        ┌──────────────────────────────────────────────────────┐
                        │                    Hub 服务器                        │
                        │                  (freekiosk-hub)                     │
                        │                                               │
                        │  ┌─────────────┐  ┌─────────────┐  ┌─────────┐  │
                        │  │  REST API   │  │  MQTT Svc   │  │  SSE    │  │
                        │  │  (Echo)     │  │             │  │         │  │
                        │  └──────┬──────┘  └──────┬──────┘  └────┬────┘  │
                        │         │                │               │        │
                        │         └────────────────┼───────────────┘        │
                        │                          │                        │
                        │  ┌──────────────────────┴──────────────────────┐  │
                        │  │              FieldTripHandler                 │  │
                        │  │  - CreateGroup / CreateDevice               │  │
                        │  │  - BindDevice / ReportLocation             │  │
                        │  │  - PollCommands / SendBroadcast            │  │
                        │  │  - SetWhitelist / OTA                      │  │
                        │  └──────────────────────┬──────────────────────┘  │
                        │                         │                          │
                        │  ┌─────────────────────┴───────────────────────┐  │
                        │  │         FieldTripRepository (SQLite)        │  │
                        │  └───────────────────────────────────────────┘  │
                        └──────────────────────────────────────────────────────┘
                                    │                          │
                    ┌───────────────┴──────────────────────────┐
                    │                                          │
              MQTT Topic                                 HTTPS REST
    fieldtrip/{group_id}/broadcast               /api/v2/fieldtrip/*
                    │                                          │
          ┌─────────┴─────────┐                    ┌─────────┴─────────┐
          │   Android 平板    │                    │   Android 平板    │
          │ ┌─────────────┐  │                    │ ┌─────────────┐  │
          │ │ MQTT Client │  │                    │ │ HTTP Client │  │
          │ │  (订阅广播) │  │                    │ │ (轮询命令)  │  │
          │ └─────────────┘  │                    │ └──────┬──────┘  │
          │                  │                    │         │          │
          │ ┌─────────────┐  │                    │  ┌─────┴─────┐   │
          │ │BroadcastOver│  │                    │  │ HubConfig  │   │
          │ │  layActivity│  │                    │  │ Module     │   │
          │ └─────────────┘  │                    │  └───────────┘   │
          │                  │                    │                   │
          │ ┌─────────────┐  │                    │ ┌─────────────┐  │
          │ │GPS Reporter │  │                    │ │ QrScanner   │  │
          │ │ (FusedLoc) │  │                    │ │ (CameraX)   │  │
          │ └─────────────┘  │                    │ └─────────────┘  │
          └──────────────────┘                    └───────────────────┘

    ┌────────────────────────────────────────────────────────────────┐
    │                      EMQX MQTT Broker                          │
    │           (fieldtrip/{group_id}/broadcast)                    │
    └────────────────────────────────────────────────────────────────┘
```

### 2.2 数据流 — 设备绑定

```
  教师在 Hub 创建分组和设备
           │
           ▼
  ┌───────────────────┐
  │ POST /groups      │  创建分组
  └─────────┬─────────┘
            │
  ┌─────────▼─────────┐
  │ POST /devices     │  创建设备 → 返回 {api_key, group_key, hub_url}
  └─────────┬─────────┘
            │
  ┌─────────▼─────────┐
  │ Hub 生成 QR 码     │  设备信息编码为 JSON → 生成二维码
  └─────────┬─────────┘
            │
  ┌─────────▼─────────┐
  │ 平板扫描 QR 码    │  CameraX + ZXing 识别
  └─────────┬─────────┘
            │
  ┌─────────▼─────────────────┐
  │ HubConfigModule.bindWithQr │  解析 QR JSON
  │ Payload()                 │  POST /api/v2/fieldtrip/devices/bind
  └─────────┬─────────────────┘
            │
  ┌─────────▼─────────┐
  │ Hub 验证 API Key   │  写入 DB: status='active', group_id
  │ 返回 BindResponse │  返回 {device_id, group_id, signing_pubkey}
  └─────────┬─────────┘
            │
  ┌─────────▼─────────┐
  │ 平板存储配置      │  SharedPreferences 保存 hub_url, api_key
  │ 开始 GPS 上报     │  FusedLocationProviderClient
  │ 开始 MQTT 订阅    │  订阅 fieldtrip/{group_id}/broadcast
  └───────────────────┘
```

### 2.3 数据流 — 活动广播

```
  教师在 Hub 输入广播消息
           │
           ▼
  ┌─────────────────────────────────┐
  │ POST /api/v2/fieldtrip/broadcast │  保存到 broadcasts 表
  └───────────────┬─────────────────┘
                  │
    ┌─────────────┴─────────────┐
    │ BroadcastService.SendToGroup │
    └─────────────┬─────────────┘
                  │
    ┌─────────────▼─────────────┐
    │ MQTTService.Publish()       │
    │ topic: fieldtrip/{group_id}/broadcast │
    │ payload: {message, sound}  │
    └─────────────┬─────────────┘
                  │
    ┌─────────────▼─────────────┐
    │ EMQX Broker 转发消息       │
    └─────────────┬─────────────┘
                  │
        ┌─────────┴─────────┐
        │   每台平板 MQTT     │
        │   Client 收到消息  │
        └─────────┬─────────┘
                  │
        ┌─────────▼─────────────────────────┐
        │ KioskMqttClient.handleBroadcast   │
        │ Message()                         │
        └─────────┬─────────────────────────┘
                  │
        ┌─────────▼─────────────────────────┐
        │ 启动 BroadcastOverlayActivity      │  全屏覆盖层
        │ 显示消息 + 播放提示音              │  10 秒后自动消失
        └─────────────────────────────────┘
```

---

## 3. Hub 服务器组件

### 3.1 项目结构

```
freekiosk-hub/
├── cmd/server/main.go           # 程序入口
├── internal/
│   ├── api/
│   │   ├── router.go           # Echo 路由配置
│   │   ├── fieldtrip_handler.go # Field Trip API 处理器
│   │   ├── fieldtrip_ui.go    # Field Trip Web UI 处理器
│   │   └── ota_handler.go     # OTA 升级处理器
│   ├── models/
│   │   └── fieldtrip.go       # 数据模型
│   ├── repositories/
│   │   └── fieldtrip_repo.go  # 数据库访问层
│   ├── services/
│   │   └── broadcast_service.go # 广播服务
│   └── mqtt/
│       ├── client.go          # MQTT 5.0 客户端
│       └── config.go          # MQTT 配置
├── ui/                        # Web UI 模板 (Templ)
│   └── fieldtrip.templ
└── docs/field-trip/           # 本文档
```

### 3.2 路由一览

| 方法 | 路径 | 处理器 | 说明 |
|------|------|--------|------|
| GET | `/health` | SystemJSONHandler | 健康检查 |
| GET | `/health/mqtt` | — | MQTT 连接状态 |
| GET | `/sse/global` | — | 全局 SSE 推送 |
| GET | `/sse/tablet/:id` | — | 单设备 SSE 推送 |
| GET | `/api/v2/ws` | WebSocketHandler | WebSocket 端点 |
| GET | `/api/v2/ws/connections` | WebSocketHandler | 连接数统计 |
| POST | `/api/v2/fieldtrip/groups` | FieldTripHandler | 创建分组 |
| GET | `/api/v2/fieldtrip/groups` | FieldTripHandler | 列出分组 |
| DELETE | `/api/v2/fieldtrip/groups/:id` | FieldTripHandler | 删除分组 |
| POST | `/api/v2/fieldtrip/devices` | FieldTripHandler | 创建设备 |
| GET | `/api/v2/fieldtrip/devices` | FieldTripHandler | 列出设备 |
| DELETE | `/api/v2/fieldtrip/devices/:id` | FieldTripHandler | 删除设备 |
| PATCH | `/api/v2/fieldtrip/devices/:id` | FieldTripHandler | 更新设备 |
| POST | `/api/v2/fieldtrip/devices/bind` | FieldTripHandler | 设备绑定确认 |
| POST | `/api/v2/fieldtrip/devices/:id/location` | FieldTripHandler | GPS 上报 |
| GET | `/api/v2/fieldtrip/devices/:id/location/history` | FieldTripHandler | GPS 历史 |
| POST | `/api/v2/fieldtrip/devices/:id/whitelist` | FieldTripHandler | 设置白名单 |
| GET | `/api/v2/fieldtrip/commands` | FieldTripHandler | 轮询命令 |
| POST | `/api/v2/fieldtrip/broadcast` | FieldTripHandler | 发送广播 |
| POST | `/api/v2/fieldtrip/ota/upload` | OTAHandler | 上传 APK |
| GET | `/api/v2/fieldtrip/ota/list` | OTAHandler | OTA 版本列表 |
| GET | `/fieldtrip` | FieldTripUIHandler | 研学版管理页面 |
| GET | `/fieldtrip/groups/new` | FieldTripUIHandler | 新建分组页面 |
| GET | `/fieldtrip/groups/:id/edit` | FieldTripUIHandler | 编辑分组页面 |

### 3.3 关键服务

#### BroadcastService

```go
type BroadcastService struct {
    repo     *FieldTripRepository
    mqttSvc  *MQTTService  // 可能为 nil（降级模式）
}

func (s *BroadcastService) SendToGroup(groupID, message, sound string) error
// 将消息写入 broadcasts 表，如果 MQTT 可用则通过 MQTT 发布
```

**MQTT 降级逻辑：**
- `mqttSvc == nil` → 仅写入 DB，不发布
- `mqttSvc.IsConnected() == false` → 仅写入 DB，不发布
- 设备通过 HTTP 轮询 `/commands` 获取广播（降级路径）

#### FieldTripHandler

```go
type FieldTripHandler struct {
    Repo     *FieldTripRepository
    BcastSvc *BroadcastService  // 通过路由传入
}
// 15 个 API 端点实现
```

---

## 4. Android 平板应用组件

### 4.1 新增模块

| 模块文件 | 职责 | 关键类 |
|---------|------|--------|
| `QrScannerModule.kt` | CameraX + ZXing QR 扫描 | `scanQr()`, `stopScanning()` |
| `HubConfigModule.kt` | Hub 配置存储 + GPS 上报 | `getConfig()`, `bindWithQrPayload()`, `startGpsReporting()` |
| `BroadcastOverlayActivity.kt` | 全屏广播通知 | 10s 自动消失，播放提示音 |
| `AppWhitelistManager.kt` | 应用白名单检查 | `isAppAllowed()` |
| `KioskMqttClient.kt` | MQTT 广播订阅 | `subscribeToFieldtripBroadcast()` |
| `MqttModule.kt` | MQTT React Native 桥接 | `setGroupId()` |
| `AppLauncherModule.kt` | 集成白名单检查 | 修改 `launchExternalApp()` |
| `KioskHttpServer.kt` | Field Trip REST 路由 | 5 个路由 |

### 4.2 依赖项

```gradle
// build.gradle (app)
dependencies {
    // CameraX QR 扫描
    def camerax_version = "1.3.1"
    implementation "androidx.camera:camera-core:$camerax_version"
    implementation "androidx.camera:camera-camera2:$camerax_version"
    implementation "androidx.camera:camera-lifecycle:$camerax_version"
    implementation "androidx.camera:camera-view:$camerax_version"

    // ZXing 二维码
    implementation "com.google.zxing:core:3.5.2"

    // Google Play Services Location (GPS)
    implementation "com.google.android.gms:play-services-location:21.1.0"
}
```

### 4.3 绑定流程时序

```
┌────────────┐         ┌───────────────┐       ┌────────────┐      ┌─────────┐
│   JS UI    │         │ QrScanner    │       │ HubConfig  │      │  Hub    │
│            │         │ Module       │       │ Module     │      │ Server  │
└─────┬──────┘         └──────┬──────┘       └──────┬─────┘      └────┬────┘
      │  scanQr()                │                   │                 │
      │─────────────────────────>│                   │                 │
      │                          │ 打开相机，开始扫描 │                 │
      │                          │                   │                 │
      │                          │ <QR 数据>         │                 │
      │<─────────────────────────│                   │                 │
      │                          │                   │                 │
      │ bindWithQrPayload(qrData)                   │                 │
      │────────────────────────────────────────────>│                 │
      │                          │                   │ POST /bind      │
      │                          │                   │───────────────>│
      │                          │                   │   {success}    │
      │                          │                   │<───────────────│
      │                          │                   │                 │
      │                          │                   │ SharedPrefs    │
      │<────────────────────────────────────────────│                 │
      │                          │                   │                 │
      │                          │                   │ startGpsReporting()
      │                          │                   │────────────────>│ POST /location
```

---

## 5. API 接口规范

### 5.1 认证方式

Field Trip API 使用 `X-Api-Key` Header 进行设备认证。

```
X-Api-Key: <device_api_key>
```

**注意：** Header 名称是 `X-Api-Key`（不是 `X-API-Key`），大小写敏感。Android 客户端和服务器必须保持一致。

### 5.2 设备绑定

**POST** `/api/v2/fieldtrip/devices/bind`

请求：
```json
{
  "device_id": "设备ID（UUID）",
  "group_key": "分组密钥（教师提供的字符串）",
  "api_key": "API密钥（教师提供的字符串）",
  "hub_url": "Hub服务器地址"
}
```

响应（200）：
```json
{
  "device_id": "设备ID",
  "group_id": "分组ID",
  "signing_pubkey": "签名公钥（Ed25519）",
  "broadcast_sound": "default",
  "update_policy": "manual"
}
```

错误响应：
- `401` — API Key 不匹配
- `404` — 设备 ID 或分组密钥不存在

### 5.3 GPS 上报

**POST** `/api/v2/fieldtrip/devices/:id/location`

请求：
```json
{
  "device_id": "设备ID",
  "lat": 31.2304,
  "lng": 121.4737,
  "accuracy": 10.0,
  "timestamp": 1774150700
}
```

响应：**204 No Content**（数据存储成功，无响应体）

### 5.4 命令轮询

**GET** `/api/v2/fieldtrip/commands?device_id=<device_id>`

响应（200，无待处理命令）：
```json
{}
```

响应（有待处理命令）：
```json
{
  "whitelist": ["com.android.chrome", "com.android.settings"],
  "ota_url": "http://hub.example.com/apk/v2.apk",
  "broadcast": "集合时间到了！"
}
```

### 5.5 白名单设置

**POST** `/api/v2/fieldtrip/devices/:id/whitelist`

请求：
```json
{
  "apps": ["com.android.chrome", "com.android.settings"]
}
```

响应（200）：
```json
{
  "status": "pending"
}
```

### 5.6 广播

**POST** `/api/v2/fieldtrip/broadcast`

请求：
```json
{
  "group_id": "分组ID",
  "message": "集合时间到了！",
  "sound": "chime"
}
```

响应（200）：
```json
{
  "id": "广播ID",
  "group_id": "分组ID",
  "message": "集合时间到了！",
  "sound": "chime",
  "created_by": "system",
  "created_at": 1774150747,
  "delivered_count": 0,
  "failed_count": 0
}
```

### 5.7 OTA 升级

**POST** `/api/v2/fieldtrip/ota/upload`

请求：`multipart/form-data`，字段 `file`（APK 文件）

响应（200）：
```json
{
  "version": "2.1.0",
  "filename": "app-release.apk",
  "size": 15728640,
  "uploaded_at": 1774150747
}
```

**GET** `/api/v2/fieldtrip/ota/list`

响应（200）：
```json
[
  {
    "version": "2.1.0",
    "filename": "app-release.apk",
    "size": 15728640,
    "uploaded_at": 1774150747
  }
]
```

---

## 6. 数据库结构

### 6.1 研学版相关表

```sql
-- 分组表
CREATE TABLE fieldtrip_groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    group_key TEXT UNIQUE NOT NULL,    -- 教师提供的绑定密钥
    broadcast_sound TEXT DEFAULT 'default',
    update_policy TEXT DEFAULT 'manual', -- 'manual' | 'auto'
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- 设备表
CREATE TABLE fieldtrip_devices (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    group_id TEXT REFERENCES fieldtrip_groups(id),
    api_key_hash TEXT NOT NULL,        -- SHA-256 哈希存储
    hub_url TEXT NOT NULL,
    last_seen INTEGER,                 -- Unix 时间戳
    last_lat REAL,                     -- 纬度
    last_lng REAL,                    -- 经度
    status TEXT DEFAULT 'active',     -- 'pending' | 'active' | 'inactive'
    signing_pubkey TEXT,               -- Ed25519 公钥
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- GPS 日志表
CREATE TABLE gps_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT REFERENCES fieldtrip_devices(id) ON DELETE CASCADE,
    lat REAL NOT NULL,
    lng REAL NOT NULL,
    accuracy REAL,
    timestamp INTEGER NOT NULL,        -- 设备上报的时间戳
    created_at INTEGER NOT NULL        -- 服务器接收时间
);
CREATE INDEX idx_gps_device ON gps_logs(device_id);
CREATE INDEX idx_gps_timestamp ON gps_logs(timestamp);

-- 广播记录表
CREATE TABLE broadcasts (
    id TEXT PRIMARY KEY,
    group_id TEXT REFERENCES fieldtrip_groups(id),
    message TEXT NOT NULL,
    sound TEXT DEFAULT 'default',
    created_by TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    delivered_count INTEGER DEFAULT 0,
    failed_count INTEGER DEFAULT 0
);

-- 待处理命令表
CREATE TABLE pending_commands (
    id TEXT PRIMARY KEY,
    device_id TEXT NOT NULL,
    command_type TEXT NOT NULL,       -- 'whitelist' | 'ota' | 'broadcast'
    payload TEXT NOT NULL,            -- JSON 格式
    status TEXT DEFAULT 'pending',   -- 'pending' | 'delivered'
    created_at INTEGER NOT NULL,
    delivered_at INTEGER
);
CREATE INDEX idx_pc_device ON pending_commands(device_id, status);
```

### 6.2 API Key 存储安全

API Key 在数据库中存储 **SHA-256 哈希值**，而非明文。验证流程：

```
客户端发送 API Key → 服务器计算 SHA-256 → 与数据库存储的哈希比对
```

### 6.3 Group Key vs API Key

| 字段 | 用途 | 生成方式 |
|------|------|---------|
| `group_key` | 设备绑定时验证分组身份 | 教师在 Hub 创建分组时自动生成（32 字符随机字符串） |
| `api_key` | 所有 API 请求的认证凭证 | 教师创建设备时自动生成（48 字符随机字符串） |
| `api_key_hash` | API Key 的 SHA-256 哈希 | 创建时自动计算存储 |

---

## 7. 功能模块详解

### 7.1 QR 快速绑定

**流程：** 教师在 Hub 创建分组和设备 → Hub 显示设备 QR 码 → 平板扫描 → 自动配置

**QR 码内容（JSON）：**
```json
{
  "device_id": "f97f96d7-e899-48da-9733-8b79ea27a9c7",
  "group_key": "dkYZsNT8SDqCNnFnuJ3CH3AHmcdlEf5OZ5gqnD1ne7Y",
  "api_key": "JEUg4vlT-eTU-20ahXUiRGHN_hyfrNMRy0UnfcOBI6s",
  "hub_url": "http://192.168.1.100:8081"
}
```

**平板端处理：** `HubConfigModule.bindWithQrPayload()` 解析 JSON → POST 到 `/api/v2/fieldtrip/devices/bind` → 保存到 SharedPreferences

**生命周期：** `LifecycleEventListener` 监听平板生命周期，切换到后台时停止 GPS，切换到前台时恢复。

### 7.2 GPS 地理位置监控

**上报频率：** 由平板端 `HubConfigModule.startGpsReporting(intervalSeconds)` 控制，默认 30 秒。

**上报内容：** 纬度、经度、精度、设备端时间戳

**Hub 存储：** 写入 `gps_logs` 表，同时更新 `fieldtrip_devices.last_lat/lng/last_seen`

**Dashboard 展示：** 通过 SSE 实时推送更新，前端地图标记设备位置

**GPS 精度过滤：** 建议忽略 `accuracy > 100` 的数据（室内 GPS 精度差）

### 7.3 活动广播（MQTT）

**发布主题：** `fieldtrip/{group_id}/broadcast`

**消息格式：**
```json
{
  "id": "广播ID",
  "message": "集合时间到了！",
  "sound": "chime",
  "created_at": 1774150747
}
```

**平板端订阅：** `KioskMqttClient.subscribeToFieldtripBroadcast(groupID)` → 收到消息 → 启动 `BroadcastOverlayActivity`

**降级方案：** MQTT 不可用时，平板通过 HTTP 轮询 `/api/v2/fieldtrip/commands` 获取广播内容（轮询间隔 30 秒）。

### 7.4 应用白名单

**工作原理：**
1. 教师在 Hub 设置设备白名单 → `pending_commands` 表写入 `whitelist` 命令
2. 平板轮询 `/api/v2/fieldtrip/commands` → 收到白名单 → 保存到 SharedPreferences
3. 学生尝试打开 App 时 → `AppLauncherModule.launchExternalApp()` → 调用 `AppWhitelistManager.isAppAllowed()` → 拒绝非白名单应用，显示 Toast

**特殊逻辑：** 如果平板未绑定 Hub（无 SharedPreferences 配置），`AppWhitelistManager.isAppAllowed()` 返回 `true`（允许所有应用，标准 Kiosk 行为）。

### 7.5 OTA 远程升级

**上传：** 教师通过 Hub UI 上传 APK → `/api/v2/fieldtrip/ota/upload` → 存储到 `apk/` 目录

**推送流程：**
1. 教师在 Hub 选择设备 → 发起 OTA 命令
2. 写入 `pending_commands` 表（类型 `ota`，payload 含 APK URL）
3. 平板轮询 `/api/v2/fieldtrip/commands` → 收到 OTA 命令
4. 平板下载 APK → 验证签名 → 安装 → 重启

**APK 签名：** 研学版使用 Ed25519 签名验证（未来支持）。当前版本 APK 存放在 `apk/` 目录，路径通过 URL 暴露。

---

## 8. 部署指南

### 8.1 环境要求

| 组件 | 要求 |
|------|------|
| Hub 服务器 | Linux/macOS，Go 1.25+，SQLite |
| MQTT Broker | EMQX 5.1+（可选，HTTP 降级可用） |
| Android 平板 | Android 8.0+，已注册为 Device Owner |
| 网络 | Hub 和平板在同一网络，或平板可访问 Hub 公网地址 |

### 8.2 Hub 部署

**方式一：Docker Compose（推荐）**

```bash
cd freekiosk-hub
docker-compose up -d
```

**方式二：直接运行**

```bash
cd freekiosk-hub
go build -o bin/freekiosk-hub cmd/server/main.go

# 运行
SERVER_PORT=8081 \
DB_PATH=freekiosk.db \
TS_AUTHKEY=tskey-xxx \
KIOSK_API_KEY=your-secret-key \
MQTT_BROKER_URL=tcp://localhost:1883 \
./bin/freekiosk-hub
```

### 8.3 关键配置项

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `SERVER_PORT` | `8081` | Web 服务端口 |
| `DB_PATH` | `freekiosk.db` | SQLite 数据库路径 |
| `TS_AUTHKEY` | — | Tailscale API 密钥（可选，用于零接触内网穿透） |
| `KIOSK_API_KEY` | — | 平板认证密钥 |
| `KIOSK_PORT` | `8080` | 平板 API 端口 |
| `MQTT_BROKER_URL` | `localhost` | MQTT Broker 地址 |
| `MQTT_PORT` | `1883` | MQTT Broker 端口 |
| `MQTT_USE_TLS` | `false` | 是否启用 TLS |
| `POLL_INTERVAL` | `30s` | 平板状态轮询间隔 |

### 8.4 Docker Compose 配置

```yaml
version: '3.8'
services:
  freekiosk-hub:
    build: .
    ports:
      - "8081:8081"
    environment:
      - SERVER_PORT=8081
      - DB_PATH=/data/freekiosk.db
      - TS_AUTHKEY=${TS_AUTHKEY}
      - KIOSK_API_KEY=${KIOSK_API_KEY}
      - MQTT_BROKER_URL=emqx
      - MQTT_PORT=1883
    volumes:
      - ./data:/data
      - ./apk:/apk
    depends_on:
      - emqx
    restart: unless-stopped

  emqx:
    image: emqx/emqx:5.1.0
    ports:
      - "1883:1883"
      - "8083:8083"
      - "18083:18083"
    restart: unless-stopped
```

---

## 9. 运维指南

### 9.1 健康检查

```bash
# HTTP 健康检查
curl http://localhost:8081/health
# {"status":"ok","database":"connected","version":"0.0.2"}

# MQTT 连接状态
curl http://localhost:8081/health/mqtt
# {"status":"connected","message":"MQTT 服务正常运行"}
# 或
# {"status":"disconnected","message":"MQTT 未连接到 Broker"}
```

### 9.2 日志分析

Hub 使用结构化日志（slog），关键日志关键词：

| 关键词 | 含义 |
|--------|------|
| `[MQTT] 连接错误` | MQTT Broker 连接失败 |
| `resource created: new fieldtrip group added` | 分组创建成功 |
| `resource created: device registered` | 设备注册成功 |
| `Broadcast published to MQTT` | 广播已发布 |
| `GPS report received` | GPS 数据上报 |

### 9.3 Prometheus 指标

访问 `http://localhost:8081/metrics` 获取 Prometheus 格式指标：

| 指标名 | 类型 | 说明 |
|--------|------|------|
| `freekiosk_devices_total` | Gauge | 设备总数 |
| `freekiosk_devices_active` | Gauge | 在线设备数 |
| `freekiosk_mqtt_messages_total` | Counter | MQTT 消息总数 |
| `freekiosk_api_requests_total` | Counter | API 请求总数 |
| `freekiosk_commands_total` | Counter | 命令下发总数 |

### 9.4 数据库维护

```bash
# 查看设备数量
sqlite3 freekiosk.db "SELECT COUNT(*) FROM fieldtrip_devices;"

# 查看分组列表
sqlite3 freekiosk.db "SELECT id, name, group_key FROM fieldtrip_groups;"

# 查看 GPS 日志条数
sqlite3 freekiosk.db "SELECT COUNT(*) FROM gps_logs;"

# 查看广播记录
sqlite3 freekiosk.db "SELECT id, group_id, message, delivered_count FROM broadcasts;"

# 清理 30 天前的 GPS 日志
sqlite3 freekiosk.db "DELETE FROM gps_logs WHERE created_at < strftime('%s','now') - 86400 * 30;"
```

### 9.5 常见故障排查

| 症状 | 可能原因 | 解决方案 |
|------|---------|---------|
| 平板绑定失败 | group_key 不正确 | 确认教师提供的 group_key 与 Hub 一致 |
| GPS 不上报 | 平板未授予定位权限 | 检查 Android 定位权限 |
| 广播收不到 | MQTT Broker 不可用 | 检查 EMQX 服务状态，验证 MQTT 连接 |
| OTA 安装失败 | APK 签名验证失败 | 检查 APK 是否正确签名 |
| API 返回 404 | Hub 未启动 | 检查 Hub 进程是否运行 |
| Web UI 空白 | Templ 组件未生成 | 运行 `make generate` 重新生成 |

---

## 10. 常见问题

### Q1: 平板扫描 QR 码后绑定失败，提示 "group not found"

**原因：** 平板输入的 `group_key` 与 Hub 中存储的不匹配。

**排查：**
1. 在 Hub 的研学版管理页面检查分组的 `group_key`
2. 确认平板扫描的 QR 码是正确分组下的设备

### Q2: 广播消息发送成功，但平板没有收到

**原因：** MQTT Broker 不可用（降级到 HTTP 轮询模式）。

**排查：**
1. 检查 `curl http://localhost:8081/health/mqtt` 是否显示 `connected`
2. 如果是 `disconnected`，检查 EMQX 是否运行
3. 平板会在 30 秒内通过轮询 `/api/v2/fieldtrip/commands` 获取广播

### Q3: GPS 位置在 Hub 显示不正确（偏移很大）

**原因：** GPS 精度不足（室内或信号弱）。

**排查：** 检查平板 GPS 精度值，`accuracy > 100` 的数据建议过滤。

### Q4: OTA 推送后平板没有反应

**原因：** 平板可能处于离线状态，或轮询间隔过长。

**排查：**
1. 检查平板是否在线（Hub Dashboard 设备列表）
2. 确认平板的轮询间隔设置
3. 手动触发平板调用 `/api/v2/fieldtrip/commands` 检查待处理命令

### Q5: 如何重置平板的 Hub 配置？

平板清除配置的方式：在平板上卸载并重新安装 FreeKiosk App，SharedPreferences 会被清除。

### Q6: Hub 支持多少台平板同时管理？

理论上无硬性限制。SQLite 在 1000 台设备级别无明显性能问题。实际限制因素：
- MQTT Broker 的并发连接数
- GPS 上报频率（建议间隔 ≥ 30 秒）
- SSE 连接数（每个浏览器 tab 一个）

---

## 附录 A：HTTP API 完整列表

```
基础健康
  GET  /health
  GET  /health/mqtt

研学版 API (需 X-Api-Key 认证)
  分组管理
    POST   /api/v2/fieldtrip/groups
    GET    /api/v2/fieldtrip/groups
    DELETE /api/v2/fieldtrip/groups/:id

  设备管理
    POST   /api/v2/fieldtrip/devices
    GET    /api/v2/fieldtrip/devices
    DELETE /api/v2/fieldtrip/devices/:id
    PATCH  /api/v2/fieldtrip/devices/:id

  设备绑定
    POST   /api/v2/fieldtrip/devices/bind

  GPS 监控
    POST   /api/v2/fieldtrip/devices/:id/location
    GET    /api/v2/fieldtrip/devices/:id/location/history

  命令和广播
    POST   /api/v2/fieldtrip/devices/:id/whitelist
    GET    /api/v2/fieldtrip/commands
    POST   /api/v2/fieldtrip/broadcast

  OTA 升级
    POST   /api/v2/fieldtrip/ota/upload
    GET    /api/v2/fieldtrip/ota/list

实时通信
  GET  /sse/global
  GET  /sse/tablet/:id
  GET  /api/v2/ws
  GET  /api/v2/ws/connections
```

---

## 附录 B：相关文件路径

### Hub (Go)

| 路径 | 说明 |
|------|------|
| `cmd/server/main.go` | 入口，初始化所有服务 |
| `internal/api/router.go` | 路由注册 |
| `internal/api/fieldtrip_handler.go` | Field Trip API 处理器 |
| `internal/api/fieldtrip_ui.go` | Web UI 处理器 |
| `internal/api/ota_handler.go` | OTA 处理器 |
| `internal/models/fieldtrip.go` | 数据模型 |
| `internal/repositories/fieldtrip_repo.go` | 数据库访问层 |
| `internal/services/broadcast_service.go` | 广播服务 |
| `internal/mqtt/client.go` | MQTT 客户端封装 |
| `ui/fieldtrip.templ` | Field Trip UI 模板 |

### Android (Kotlin)

| 路径 | 说明 |
|------|------|
| `android/app/src/main/java/com/freekiosk/QrScannerModule.kt` | QR 扫描 |
| `android/app/src/main/java/com/freekiosk/HubConfigModule.kt` | Hub 配置和 GPS |
| `android/app/src/main/java/com/freekiosk/BroadcastOverlayActivity.kt` | 广播通知界面 |
| `android/app/src/main/java/com/freekiosk/AppWhitelistManager.kt` | 白名单管理 |
| `android/app/src/main/java/com/freekiosk/mqtt/KioskMqttClient.kt` | MQTT 客户端 |
| `android/app/src/main/java/com/freekiosk/AppLauncherModule.kt` | 应用启动器 |

---

*本文档由 Claude Code 自动生成，最后更新于 2026-03-22*
