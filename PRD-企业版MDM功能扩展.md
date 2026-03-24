# FreeKiosk 企业版 MDM 功能扩展 PRD

## 1. 产品概述

### 1.1 背景与目标

FreeKiosk Hub 研学版已实现基础的设备管理和 Field Trip 功能。基于 Headwind MDM API 的全面分析，本 PRD 旨在将 FreeKiosk Hub 打造成功能完整的企业级移动设备管理（MDM）系统，同时在 Android 客户端（FreeKiosk）实现与之配套的设备端功能。

### 1.2 核心价值主张

| 维度 | 现状 | 目标 |
|------|------|------|
| 设备管理 | 基础注册、分组、GPS | 完整生命周期管理 |
| 应用管理 | 无 | 企业应用商店、静默安装 |
| 配置管理 | 基础 Field Trip | 策略配置、配置文件 |
| 远程控制 | 命令（beep/reboot） | 完整远程控制、会话管理 |
| 安全控制 | 基础白名单 | 网络过滤、流量监控 |
| 用户管理 | 单租户 | 多租户、RBAC、LDAP |

---

## 2. 功能模块规划

### Phase 1: 设备管理核心 (P0)

#### 2.1.1 设备注册与生命周期管理

**功能列表：**
- [x] 设备二维码注册（已有 Field Trip）
- [ ] 标准设备注册 API（支持更多元化注册方式）
- [ ] 设备信息同步（model, IMEI, phone, carrier）
- [ ] 设备分组管理（支持多级分组）
- [ ] 设备标签和自定义属性
- [ ] 批量设备导入/导出（CSV/Excel）
- [ ] 设备注销和重新注册

**技术方案：**
```
Hub 端：
- 扩展 FieldTripDevice 模型，添加更多设备属性
- 新增 Device Model 和 DeviceRepository
- 支持设备唯一标识（number/IMEI）
- 实现设备搜索和过滤 API

Android 端：
- 扩展设备信息上报协议
- 支持设备主动注册和心跳
- 实现设备属性动态更新
```

#### 2.1.2 远程设备控制

**功能列表：**
- [x] 设备锁定 (lock)
- [x] 设备重启 (reboot)
- [x] 恢复出厂设置 (factory reset)
- [ ] 设备响铃/报警
- [ ] 远程屏幕截图
- [ ] 远程屏幕直播/观看
- [ ] 远程输入（键盘/鼠标）
- [ ] 设备 GPS 定位（立即定位）
- [ ] 设备时间同步

**技术方案：**
```
Hub → MQTT → Android：
{
  "id": "cmd-xxx",
  "type": "screenshot|location|input|...",
  "params": {...},
  "timeout": 30
}

Android → Hub：
{
  "commandId": "cmd-xxx",
  "success": true,
  "result": "base64截图或GPS坐标",
  "timestamp": 1234567890
}
```

---

### Phase 2: 应用管理 (P0)

#### 2.2.1 企业应用商店

**功能列表：**
- [ ] 应用上传和管理（APK）
- [ ] 应用版本管理
- [ ] 应用分类和搜索
- [ ] 应用权限预览
- [ ] 应用评分和评论（企业内）

**技术方案：**
```
Hub 端：
- 新增 Application Model 和 Repository
- 新增 /api/v2/applications 路由
- 文件存储：本地 apk/ 目录或对象存储
- 支持 APP bundle 和 Split APK

Android 端：
- 扩展 MQTT 命令：installApp, uninstallApp, updateApp
- 实现静默安装（Device Owner 权限）
- 应用状态同步上报
```

#### 2.2.2 应用配置分发

**功能列表：**
- [ ] 应用配置模板
- [ ] 按设备/分组下发配置
- [ ] 应用初始化参数
- [ ] 应用白名单/黑名单

**技术方案：**
```
Configuration Model：
- id, name, description
- appConfigs: []AppConfig
- policies: []Policy

AppConfig：
- packageName
- configData (JSON)
- versionRange
```

---

### Phase 3: 配置与策略管理 (P0)

#### 2.3.1 设备配置文件

**功能列表：**
- [ ] 创建/编辑配置文件
- [ ] 配置文件版本管理
- [ ] 密码策略（设备解锁密码）
- [ ] 系统更新策略（立即/定时/延迟）
- [ ] 状态栏锁定
- [ ] 导航栏锁定
- [ ] 电源管理（禁用关机）

**技术方案：**
```
Configuration Schema：
{
  "id": 1,
  "name": "Kiosk Mode Standard",
  "password": "MD5_hash",
  "blockStatusBar": true,
  "blockNavigationBar": true,
  "systemUpdateType": 1, // 0=Default, 1=Immediately, 2=Scheduled, 3=Postponed
  "updateSchedule": {...},
  "deviceOwnerPackage": "com.freekiosk",
  "applications": [...],
  "files": [...]
}
```

#### 2.3.2 白名单与黑名单

**功能列表：**
- [ ] 应用白名单（仅允许安装指定应用）
- [ ] 应用黑名单（禁止指定应用）
- [ ] 网站白名单（浏览器网址过滤）
- [ ] 网络过滤规则

**技术方案：**
```
WhitelistPolicy：
- type: "app" | "website" | "network"
- action: "allow" | "block"
- entries: ["com.android.chrome", "*.example.com"]
```

---

### Phase 4: 位置服务与围栏 (P1)

#### 2.4.1 GPS 追踪

**功能列表：**
- [x] 设备位置上报（已有 Field Trip）
- [ ] 实时位置追踪模式
- [ ] 位置历史查询
- [ ] 位置数据导出

**技术方案：**
```
扩展 Field Trip GPS 上报协议：
- 后台定时上报（可选配置）
- 进入地理围栏时立即上报
- 电量低时降低上报频率
```

#### 2.4.2 地理围栏

**功能列表：**
- [ ] 创建/编辑地理围栏（圆形/多边形）
- [ ] 围栏命名和描述
- [ ] 进入/离开事件通知
- [ ] 围栏设备分配
- [ ] 围栏历史记录

**技术方案：**
```
Geofence Model：
{
  "id": 1,
  "name": "校园围栏",
  "latitude": 39.9042,
  "longitude": 116.4074,
  "radius": 500, // 米
  "enterNotification": true,
  "exitNotification": true,
  "active": true,
  "deviceIds": [1, 2, 3]
}
```

---

### Phase 5: 远程控制与屏幕共享 (P1)

#### 2.5.1 远程控制会话

**功能列表：**
- [ ]发起远程控制请求
- [ ]设备端同意/拒绝
- [ ] WebRTC 屏幕共享
- [ ] 远程键盘输入
- [ ] 远程点击/滑动
- [ ] 会话记录和审计

**技术方案：**
```
远程控制架构：
1. Hub 发起会话请求 → MQTT → Android
2. Android 用户同意 → Hub 建立 WebRTC 信令
3. Android 捕获屏幕 → WebRTC → Hub 浏览器
4. Hub 用户操作 → WebRTC → Android 注入事件

Session States: pending → connecting → connected → disconnected → closed
```

#### 2.5.2 屏幕截图

**功能列表：**
- [ ] 立即截图
- [ ] 定期截图
- [ ] 截图历史

---

### Phase 6: 消息推送 (P1)

#### 2.6.1 推送消息

**功能列表：**
- [ ] 发送文本消息到设备
- [ ] 消息队列和离线存储
- [ ] 消息状态确认
- [ ] 定时消息
- [ ] 批量消息推送

**技术方案：**
```
PushMessage Model：
{
  "id": "msg-xxx",
  "deviceNumber": "device-001",
  "messageType": "text|notification|command",
  "payload": "消息内容或 JSON",
  "status": "pending|sent|delivered|read",
  "createdAt": 1234567890,
  "deliveredAt": 1234567891
}
```

---

### Phase 7: 用户与权限管理 (P1)

#### 2.7.1 多租户架构

**功能列表：**
- [x] 租户管理（已有 Phase 4-5）
- [ ] 租户配额管理
- [ ] 租户数据隔离

#### 2.7.2 用户角色与权限

**功能列表：**
- [ ] 用户注册和登录
- [ ] 角色定义（Admin/Operator/Viewer）
- [ ] 权限矩阵
- [ ] LDAP/AD 集成

**技术方案：**
```
UserRole Permissions：
- device:read, device:write, device:delete
- config:read, config:write
- app:read, app:write, app:install
- user:read, user:write
- admin:all

LDAP Integration：
- LDAP Server 配置
- 用户组映射
- 自动同步
```

---

### Phase 8: 日志与审计 (P2)

#### 2.8.1 设备日志

**功能列表：**
- [ ] 设备日志收集
- [ ] 日志规则配置
- [ ] 日志搜索和导出

#### 2.8.2 审计日志

**功能列表：**
- [x] 操作审计（已有部分）
- [ ] 完整审计日志查询
- [ ] 审计日志导出

**技术方案：**
```
AuditLog Model：
{
  "id": 1,
  "userId": "admin",
  "action": "device.lock",
  "targetType": "device",
  "targetId": "device-001",
  "details": {...},
  "ip": "192.168.1.1",
  "timestamp": 1234567890
}
```

---

### Phase 9: 网络安全 (P2)

#### 2.9.1 网络过滤

**功能列表：**
- [ ] 网络流量规则
- [ ] 网站过滤（DNS/URL）
- [ ] 流量统计和监控

**技术方案：**
```
NetworkRule Model：
{
  "id": 1,
  "name": "Block Social Media",
  "type": "domain|ip|url",
  "pattern": "*.facebook.com",
  "action": "block|allow|log",
  "priority": 1,
  "enabled": true
}
```

---

### Phase 10: 高级功能 (P2)

#### 2.10.1 设备照片

**功能列表：**
- [ ] 远程拍照请求
- [ ] 照片上传和查看
- [ ] 照片管理

#### 2.10.2 通讯录同步

**功能列表：**
- [ ] 通讯录管理
- [ ] 设备通讯录同步
- [ ] LDAP 通讯录集成

#### 2.10.3 白标定制

**功能列表：**
- [ ] Logo 和品牌色
- [ ] 自定义邮件模板

---

## 3. 技术架构

### 3.1 Hub 端架构

```
freekiosk-hub/
├── cmd/server/main.go
├── internal/
│   ├── api/
│   │   ├── router.go              # 路由管理
│   │   ├── device.go             # 设备管理 API (NEW)
│   │   ├── application.go        # 应用管理 API (NEW)
│   │   ├── configuration.go      # 配置管理 API (NEW)
│   │   ├── geofence.go          # 地理围栏 API (NEW)
│   │   ├── remote_control.go     # 远程控制 API (NEW)
│   │   ├── messaging.go         # 消息推送 API (NEW)
│   │   ├── user.go              # 用户管理 API (NEW)
│   │   ├── audit.go             # 审计日志 API (扩展)
│   │   └── policy.go             # 策略管理 API (扩展)
│   ├── models/
│   │   ├── device.go            # 设备模型 (NEW)
│   │   ├── application.go       # 应用模型 (NEW)
│   │   ├── configuration.go      # 配置模型 (NEW)
│   │   ├── geofence.go          # 围栏模型 (NEW)
│   │   ├── remote_session.go    # 远程会话模型 (NEW)
│   │   ├── push_message.go      # 推送消息模型 (NEW)
│   │   └── user.go              # 用户模型 (扩展)
│   ├── repositories/
│   │   ├── device_repo.go       # (NEW)
│   │   ├── application_repo.go  # (NEW)
│   │   ├── config_repo.go       # (NEW)
│   │   ├── geofence_repo.go     # (NEW)
│   │   └── user_repo.go         # (NEW)
│   └── services/
│       ├── device_service.go    # (NEW)
│       ├── app_service.go       # (NEW)
│       ├── config_service.go    # (NEW)
│       ├── geofence_service.go  # (NEW)
│       ├── remote_control_svc.go # (NEW)
│       └── mqtt_handler.go      # MQTT 命令分发 (扩展)
├── ui/
│   ├── dashboard.templ          # 仪表盘
│   ├── devices.templ           # 设备列表 (NEW)
│   ├── device_detail.templ     # 设备详情 (NEW)
│   ├── applications.templ      # 应用管理 (NEW)
│   ├── configurations.templ    # 配置管理 (NEW)
│   ├── geofences.templ         # 地理围栏 (NEW)
│   ├── remote_control.templ     # 远程控制 (NEW)
│   ├── users.templ              # 用户管理 (NEW)
│   └── settings.templ          # 系统设置
└── apk/                         # 应用存储
```

### 3.2 Android 端架构

```
freekiosk/android/app/src/main/java/com/freekiosk/
├── KioskModule.kt              # 设备 Owner API
├── api/
│   └── KioskHttpServer.kt      # 已有
├── mqtt/
│   └── KioskMqttClient.kt      # 已有，扩展命令类型
├── command/
│   ├── CommandHandler.kt       # 扩展命令处理
│   └── CommandExecutor.kt      # 命令执行
└── services/
    ├── DeviceInfoService.kt    # 设备信息服务 (NEW)
    ├── AppInstallService.kt    # 应用安装服务 (NEW)
    ├── LocationService.kt     # 位置服务 (扩展)
    ├── RemoteControlService.kt # 远程控制服务 (NEW)
    └── ConfigApplyService.kt  # 配置应用服务 (NEW)
```

### 3.3 数据模型

```
Device
├── id (PK)
├── number (unique device identifier)
├── name
├── description
├── imei
├── phone
├── model
├── manufacturer
├── osVersion
├── lastSync
├── status (active/inactive/lost)
├── configurationId (FK)
├── groupId (FK)
├── tenantId (FK)
├── metadata (JSON)
└── createdAt, updatedAt

Application
├── id (PK)
├── name
├── packageName (unique)
├── version
├── versionCode
├── apkPath
├── iconPath
├── permissions (JSON)
├── installType (kiosk|required|optional)
└── createdAt

Configuration
├── id (PK)
├── name
├── description
├── password (admin password hash)
├── settings (JSON: blockStatusBar, systemUpdateType, etc.)
├── applications (JSON array)
├── files (JSON array)
├── version
└── createdAt, updatedAt

DeviceGroup
├── id (PK)
├── name
├── parentId (self-ref for hierarchy)
├── description
└── tenantId (FK)

Geofence
├── id (PK)
├── name
├── latitude
├── longitude
├── radius
├── enterNotification
├── exitNotification
├── active
└── tenantId (FK)

RemoteSession
├── id (PK)
├── deviceId (FK)
├── userId (FK)
├── status (pending/connecting/connected/disconnected)
├── webrtcData (JSON)
├── startedAt
└── endedAt

PushMessage
├── id (PK)
├── deviceId (FK)
├── messageType
├── payload (JSON)
├── status
├── createdAt
├── deliveredAt
└── readAt

User
├── id (PK)
├── username
├── email
├── passwordHash
├── roleId (FK)
├── tenantId (FK)
├── allDevicesAvailable
├── allConfigsAvailable
└── createdAt

UserRole
├── id (PK)
├── name
├── permissions (JSON array)
└── description
```

---

## 4. API 规格

### 4.1 设备管理 API

```
GET    /api/v2/devices                    # 列表设备
POST   /api/v2/devices                    # 创建设备
GET    /api/v2/devices/:id                # 设备详情
PUT    /api/v2/devices/:id                # 更新设备
DELETE /api/v2/devices/:id                # 删除设备
POST   /api/v2/devices/search             # 搜索设备
POST   /api/v2/devices/bulk               # 批量操作
GET    /api/v2/devices/:id/location       # 获取设备位置
POST   /api/v2/devices/:id/lock           # 锁定设备
POST   /api/v2/devices/:id/reboot        # 重启设备
POST   /api/v2/devices/:id/factory-reset  # 恢复出厂
POST   /api/v2/devices/:id/screenshot    # 截图
```

### 4.2 应用管理 API

```
GET    /api/v2/applications               # 列表应用
POST   /api/v2/applications               # 上传应用
GET    /api/v2/applications/:id           # 应用详情
PUT    /api/v2/applications/:id           # 更新应用
DELETE /api/v2/applications/:id           # 删除应用
GET    /api/v2/applications/:id/versions  # 版本列表
POST   /api/v2/applications/:id/install   # 安装到设备
```

### 4.3 配置管理 API

```
GET    /api/v2/configurations             # 列表配置
POST   /api/v2/configurations             # 创建配置
GET    /api/v2/configurations/:id         # 配置详情
PUT    /api/v2/configurations/:id         # 更新配置
DELETE /api/v2/configurations/:id         # 删除配置
POST   /api/v2/configurations/:id/assign # 分配到设备/分组
```

### 4.4 地理围栏 API

```
GET    /api/v2/geofences                  # 列表围栏
POST   /api/v2/geofences                  # 创建围栏
GET    /api/v2/geofences/:id              # 围栏详情
PUT    /api/v2/geofences/:id              # 更新围栏
DELETE /api/v2/geofences/:id              # 删除围栏
GET    /api/v2/geofences/:id/devices      # 围栏内设备
POST   /api/v2/geofences/:id/devices     # 分配设备到围栏
```

### 4.5 远程控制 API

```
POST   /api/v2/remote-control/sessions     # 创建会话
GET    /api/v2/remote-control/sessions/:id # 会话详情
DELETE /api/v2/remote-control/sessions/:id # 结束会话
POST   /api/v2/remote-control/sessions/:id/connect    # 建立连接
POST   /api/v2/remote-control/sessions/:id/disconnect # 断开连接
```

### 4.6 消息推送 API

```
POST   /api/v2/messages                    # 发送消息
GET    /api/v2/messages/:id               # 消息详情
GET    /api/v2/messages                   # 消息列表
DELETE /api/v2/messages/:id              # 删除消息
```

### 4.7 用户管理 API

```
GET    /api/v2/users                       # 列表用户
POST   /api/v2/users                       # 创建用户
GET    /api/v2/users/:id                   # 用户详情
PUT    /api/v2/users/:id                   # 更新用户
DELETE /api/v2/users/:id                   # 删除用户
GET    /api/v2/roles                       # 列表角色
POST   /api/v2/roles                       # 创建角色
```

---

## 5. 实现优先级与里程碑

### Phase 1: MVP (8-12 周)

| 周 | 目标 | 交付物 |
|----|------|--------|
| 1-2 | 基础设备管理框架 | Device Model, Repository, API |
| 3-4 | 设备注册和同步 | 二维码注册, 设备信息上报 |
| 5-6 | 远程命令扩展 | lock/reboot/screenshot/location |
| 7-8 | 应用管理 MVP | 应用上传, 版本管理, 安装命令 |
| 9-10 | 配置管理 MVP | Configuration Model, 分配机制 |
| 11-12 | 测试和优化 | 集成测试, 性能优化 |

### Phase 2: 企业功能 (8-12 周)

| 周 | 目标 | 交付物 |
|----|------|--------|
| 13-14 | 地理围栏 | Geofence Model, 事件通知 |
| 15-16 | 远程控制 | WebRTC 屏幕共享 |
| 17-18 | 消息推送 | PushMessage, 离线队列 |
| 19-20 | 用户和 RBAC | User, Role, 权限矩阵 |
| 21-22 | LDAP 集成 | LDAP 配置, 自动同步 |
| 23-24 | 日志和审计 | 审计日志, 设备日志 |

### Phase 3: 高级功能 (8 周)

| 周 | 目标 | 交付物 |
|----|------|--------|
| 25-26 | 网络过滤 | NetworkRule, 流量监控 |
| 27-28 | 通讯录同步 | Contact Sync |
| 29-30 | 设备照片 | 远程拍照 |
| 31-32 | 白标和优化 | 品牌定制 |

---

## 6. 兼容性说明

### 6.1 现有 Field Trip 兼容性

- Field Trip 作为独立模块保留
- 设备可同时存在于 Field Trip 和标准 MDM
- Field Trip API 逐步迁移到统一 /api/v2/devices
- Field Trip UI 逐步融合到统一设备管理界面

### 6.2 向后兼容性

- 现有 MQTT 命令格式保持兼容
- 现有 Field Trip 设备绑定流程保持不变
- 配置通过策略服务逐步增强

---

## 7. 风险与依赖

### 7.1 技术风险

| 风险 | 缓解方案 |
|------|----------|
| WebRTC 屏幕共享延迟 | 使用更高效的编码（H.264/H.265），考虑代理模式 |
| 大规模设备 MQTT 连接 | MQTT Bridge 集群，消息分区 |
| 应用静默安装兼容性 | 仅支持 Android 8+，Device Owner 模式 |
| 设备端存储限制 | 云端存储，应用数据定期清理 |

### 7.2 依赖项

| 依赖 | 说明 |
|------|------|
| Tailscale | 设备网络连接（已有） |
| EMQX | MQTT Broker（已有） |
| SQLite/PostgreSQL | 数据存储（已有） |
| WebRTC | 远程控制（需集成） |

---

## 8. 验收标准

### Phase 1 验收

- [ ] 设备可通过二维码注册到 Hub
- [ ] 设备信息（model, IMEI, OS version）正确同步
- [ ] 支持设备分组和搜索
- [ ] 可远程锁定、重启设备
- [ ] 可获取设备当前位置
- [ ] 可截取设备屏幕
- [ ] 可上传和管理 APK 应用
- [ ] 可远程安装/卸载应用
- [ ] 可创建设备配置并分配到设备

### Phase 2 验收

- [ ] 支持地理围栏和事件通知
- [ ] 支持远程屏幕观看
- [ ] 支持消息推送和定时任务
- [ ] 支持多用户和角色权限
- [ ] 支持 LDAP 用户同步
- [ ] 完整的审计日志

### Phase 3 验收

- [ ] 网络过滤规则生效
- [ ] 通讯录同步功能正常
- [ ] 白标定制可配置

---

## 9. 非功能需求

### 9.1 性能

- 单 Hub 支持 1000+ 设备
- MQTT 消息延迟 < 100ms
- 屏幕共享帧率 ≥ 15fps
- API 响应时间 < 200ms（95th percentile）

### 9.2 可用性

- Hub 可用性 ≥ 99.5%
- 设备离线后消息缓存 7 天
- 自动重连机制

### 9.3 安全

- 所有 API JWT 认证
- MQTT TLS 加密连接
- 设备命令时效性（防重放）
- 敏感数据加密存储

---

## 10. 未来扩展方向

1. **高级分析**：设备使用报告、能耗分析、应用使用统计
2. **自动化**：基于规则的自动操作（进入围栏自动切换配置）
3. **应用虚拟化**：远程应用流（Streaming App）
4. **零接触部署**：Zero-Touch Enrollment
5. **MDM 互联**：与其他企业系统（Intune、Jamf）集成

---

*文档版本：1.0*
*创建日期：2026-03-24*
*作者：Claude*
