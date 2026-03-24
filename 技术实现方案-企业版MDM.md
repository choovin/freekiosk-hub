# FreeKiosk 企业版 MDM 技术实现方案

## 文档信息
- **版本**: 1.0
- **日期**: 2026-03-24
- **基于**: PRD-企业版MDM功能扩展.md
- **范围**: Hub 端 (Go) + Android 端 (Kotlin/TypeScript)

---

## 架构概览

```
┌─────────────────────────────────────────────────────────────────────┐
│                        FreeKiosk Hub (Go)                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │
│  │ REST API │  │ SSE/WSS │  │  MQTT   │  │   Templ UI       │  │
│  │ /api/v2  │  │ Real-time│  │ Broker  │  │   Web Interface  │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └──────────────────┘  │
│       │             │             │                                │
│  ┌────▼─────────────▼─────────────▼────┐                         │
│  │           Service Layer               │                         │
│  │  DeviceSvc | AppSvc | ConfigSvc     │                         │
│  │  GeofenceSvc | RemoteControlSvc     │                         │
│  └─────────────────┬───────────────────┘                         │
│                    │                                             │
│  ┌─────────────────▼───────────────────┐                         │
│  │        Repository Layer             │                         │
│  │  DeviceRepo | AppRepo | ConfigRepo │                         │
│  └─────────────────┬───────────────────┘                         │
│                    │                                             │
│  ┌─────────────────▼───────────────────┐                         │
│  │          SQLite / PostgreSQL        │                         │
│  └─────────────────────────────────────┘                         │
└─────────────────────────────────────────────────────────────────────┘
                               │
                    MQTT + HTTPS (Tailscale)
                               │
┌─────────────────────────────────────────────────────────────────────┐
│                   Android Device (FreeKiosk)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │ ReactNative │  │ MQTT Client  │  │   Native Module (Kotlin)  │  │
│  │   UI/TS    │  │  Subscribe   │  │   KioskModule             │  │
│  └──────────────┘  └──────┬───────┘  └──────────────────────────┘  │
│                           │                                         │
│                    ┌──────▼───────┐                              │
│                    │ CommandHandler │                              │
│                    │  Executor      │                              │
│                    └───────────────┘                              │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 第一阶段：设备管理核心 MVP（第 1-8 周）

### Hub 端实现方案

#### 第 1-2 周：基础设备管理框架

##### 1.1.1 数据库 Schema 扩展

**文件**: `internal/databases/sqlite.go`

```sql
-- 新增 MDM 设备表（与 FieldTripDevice 并存，逐步迁移）
CREATE TABLE IF NOT EXISTS mdm_devices (
    id TEXT PRIMARY KEY,
    number TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    imei TEXT,
    phone TEXT,
    model TEXT,
    manufacturer TEXT,
    os_version TEXT,
    sdk_version INTEGER,
    app_version TEXT,
    app_version_code INTEGER,
    carrier TEXT,
    last_lat REAL,
    last_lng REAL,
    last_location_time INTEGER,
    last_seen INTEGER,
    status TEXT DEFAULT 'active', -- active, inactive, lost, retired
    configuration_id TEXT,
    group_id TEXT,
    tenant_id TEXT NOT NULL,
    metadata TEXT, -- JSON for custom attributes
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (configuration_id) REFERENCES configurations(id),
    FOREIGN KEY (group_id) REFERENCES device_groups(id)
);

-- 设备分组表（支持层级）
CREATE TABLE IF NOT EXISTS device_groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    parent_id TEXT,
    description TEXT,
    tenant_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    FOREIGN KEY (parent_id) REFERENCES device_groups(id)
);

-- 设备标签表
CREATE TABLE IF NOT EXISTS device_tags (
    id TEXT PRIMARY KEY,
    device_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    value TEXT,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (device_id) REFERENCES mdm_devices(id) ON DELETE CASCADE
);

-- 设备活动日志
CREATE TABLE IF NOT EXISTS device_events (
    id TEXT PRIMARY KEY,
    device_id TEXT NOT NULL,
    event_type TEXT NOT NULL, -- location_update, status_change, config_change, etc.
    event_data TEXT, -- JSON
    created_at INTEGER NOT NULL,
    FOREIGN KEY (device_id) REFERENCES mdm_devices(id) ON DELETE CASCADE
);

-- 索引
CREATE INDEX idx_mdm_devices_tenant ON mdm_devices(tenant_id);
CREATE INDEX idx_mdm_devices_status ON mdm_devices(status);
CREATE INDEX idx_mdm_devices_number ON mdm_devices(number);
CREATE INDEX idx_device_groups_tenant ON device_groups(tenant_id);
CREATE INDEX idx_device_events_device ON device_events(device_id);
CREATE INDEX idx_device_events_created ON device_events(created_at);
```

##### 1.1.2 Model 定义

**文件**: `internal/models/mdm_device.go` (NEW)

```go
package models

import "time"

// MDMDevice 企业级设备模型
type MDMDevice struct {
    ID                string            `json:"id" db:"id"`
    Number            string            `json:"number" db:"number"` // 设备唯一标识
    Name              string            `json:"name" db:"name"`
    Description       string            `json:"description" db:"description"`
    IMEI              string            `json:"imei" db:"imei"`
    Phone             string            `json:"phone" db:"phone"`
    Model             string            `json:"model" db:"model"`
    Manufacturer      string            `json:"manufacturer" db:"manufacturer"`
    OSVersion         string            `json:"os_version" db:"os_version"`
    SDKVersion        int              `json:"sdk_version" db:"sdk_version"`
    AppVersion        string            `json:"app_version" db:"app_version"`
    AppVersionCode    int              `json:"app_version_code" db:"app_version_code"`
    Carrier           string            `json:"carrier" db:"carrier"`
    LastLat           *float64         `json:"last_lat" db:"last_lat"`
    LastLng           *float64         `json:"last_lng" db:"last_lng"`
    LastLocationTime  *int64           `json:"last_location_time" db:"last_location_time"`
    LastSeen          *int64           `json:"last_seen" db:"last_seen"`
    Status            string            `json:"status" db:"status"` // active, inactive, lost, retired
    ConfigurationID   *string         `json:"configuration_id" db:"configuration_id"`
    GroupID           *string          `json:"group_id" db:"group_id"`
    TenantID          string            `json:"tenant_id" db:"tenant_id"`
    Metadata           string            `json:"metadata" db:"metadata"` // JSON
    CreatedAt         int64            `json:"created_at" db:"created_at"`
    UpdatedAt         int64            `json:"updated_at" db:"updated_at"`
}

// DeviceStatus 设备状态枚举
type DeviceStatus string

const (
    DeviceStatusActive   DeviceStatus = "active"
    DeviceStatusInactive DeviceStatus = "inactive"
    DeviceStatusLost    DeviceStatus = "lost"
    DeviceStatusRetired DeviceStatus = "retired"
)

// DeviceGroup 设备分组
type DeviceGroup struct {
    ID          string     `json:"id" db:"id"`
    Name        string    `json:"name" db:"name"`
    ParentID    *string   `json:"parent_id" db:"parent_id"` // 支持层级
    Description string    `json:"description" db:"description"`
    TenantID    string    `json:"tenant_id" db:"tenant_id"`
    CreatedAt   int64     `json:"created_at" db:"created_at"`
    UpdatedAt   int64     `json:"updated_at" db:"updated_at"`
}

// DeviceTag 设备标签
type DeviceTag struct {
    ID        string    `json:"id" db:"id"`
    DeviceID  string    `json:"device_id" db:"device_id"`
    Tag       string    `json:"tag" db:"tag"`
    Value     string    `json:"value" db:"value"`
    CreatedAt int64     `json:"created_at" db:"created_at"`
}

// DeviceEvent 设备事件
type DeviceEvent struct {
    ID         string    `json:"id" db:"id"`
    DeviceID   string    `json:"device_id" db:"device_id"`
    EventType  string    `json:"event_type" db:"event_type"`
    EventData  string    `json:"event_data" db:"event_data"` // JSON
    CreatedAt  int64     `json:"created_at" db:"created_at"`
}

// DeviceSearchFilter 设备搜索过滤
type DeviceSearchFilter struct {
    TenantID       string   `json:"tenant_id"`
    Status         string   `json:"status,omitempty"`
    GroupID        string   `json:"group_id,omitempty"`
    ConfigurationID string  `json:"configuration_id,omitempty"`
    Search         string   `json:"search,omitempty"` // 搜索 name, number, imei
    Tags           []string `json:"tags,omitempty"`
    HasLocation    *bool    `json:"has_location,omitempty"`
    Limit          int      `json:"limit,omitempty"`
    Offset         int      `json:"offset,omitempty"`
}
```

##### 1.1.3 Repository 实现

**文件**: `internal/repositories/mdm_device_repo.go` (NEW)

```go
package repositories

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "github.com/pseudolution/freekiosk-hub/internal/models"
)

// MDMTabletRepository MDM设备仓储接口
type MDMTabletRepository interface {
    // 基础 CRUD
    CreateDevice(device *models.MDMDevice) error
    GetDeviceByID(id string) (*models.MDMDevice, error)
    GetDeviceByNumber(number string) (*models.MDMDevice, error)
    UpdateDevice(device *models.MDMDevice) error
    DeleteDevice(id string) error

    // 搜索和列表
    ListDevices(tenantID string, limit, offset int) ([]*models.MDMDevice, int64, error)
    SearchDevices(filter *models.DeviceSearchFilter) ([]*models.MDMDevice, int64, error)

    // 分组管理
    CreateGroup(group *models.DeviceGroup) error
    UpdateGroup(group *models.DeviceGroup) error
    DeleteGroup(id string) error
    ListGroups(tenantID string) ([]*models.DeviceGroup, error)
    GetGroupDevices(groupID string) ([]*models.MDMDevice, error)

    // 标签管理
    AddTag(tag *models.DeviceTag) error
    RemoveTag(deviceID, tag string) error
    GetDeviceTags(deviceID string) ([]*models.DeviceTag, error)

    // 位置管理
    UpdateLocation(deviceID string, lat, lng float64, timestamp int64) error
    GetDeviceLocation(deviceID string) (*models.GPSData, error)

    // 设备事件
    RecordEvent(event *models.DeviceEvent) error
    GetDeviceEvents(deviceID string, limit int) ([]*models.DeviceEvent, error)

    // 批量操作
    BulkUpdateStatus(deviceIDs []string, status string) error
    BulkAssignGroup(deviceIDs []string, groupID string) error
}

// SQLiteMDMTabletRepository SQLite实现
type SQLiteMDMTabletRepository struct {
    DB *sql.DB
}

// NewSQLiteMDMTabletRepository 构造函数
func NewSQLiteMDMTabletRepository(db *sql.DB) *SQLiteMDMTabletRepository {
    return &SQLiteMDMTabletRepository{DB: db}
}

func (r *SQLiteMDMTabletRepository) CreateDevice(device *models.MDMDevice) error {
    device.ID = generateUUID()
    device.CreatedAt = time.Now().Unix()
    device.UpdatedAt = time.Now().Unix()
    if device.Status == "" {
        device.Status = string(models.DeviceStatusActive)
    }

    query := `
        INSERT INTO mdm_devices (id, number, name, description, imei, phone,
            model, manufacturer, os_version, sdk_version, app_version, app_version_code,
            carrier, status, configuration_id, group_id, tenant_id, metadata,
            created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

    _, err := r.DB.Exec(query,
        device.ID, device.Number, device.Name, device.Description,
        device.IMEI, device.Phone, device.Model, device.Manufacturer,
        device.OSVersion, device.SDKVersion, device.AppVersion, device.AppVersionCode,
        device.Carrier, device.Status, device.ConfigurationID, device.GroupID,
        device.TenantID, device.Metadata, device.CreatedAt, device.UpdatedAt)

    return err
}

func (r *SQLiteMDMTabletRepository) SearchDevices(filter *models.DeviceSearchFilter) ([]*models.MDMDevice, int64, error) {
    var conditions []string
    var args []interface{}

    conditions = append(conditions, "tenant_id = ?")
    args = append(args, filter.TenantID)

    if filter.Status != "" {
        conditions = append(conditions, "status = ?")
        args = append(args, filter.Status)
    }

    if filter.GroupID != "" {
        conditions = append(conditions, "group_id = ?")
        args = append(args, filter.GroupID)
    }

    if filter.ConfigurationID != "" {
        conditions = append(conditions, "configuration_id = ?")
        args = append(args, filter.ConfigurationID)
    }

    if filter.Search != "" {
        conditions = append(conditions, "(name LIKE ? OR number LIKE ? OR imei LIKE ?)")
        search := "%" + filter.Search + "%"
        args = append(args, search, search, search)
    }

    whereClause := strings.Join(conditions, " AND ")

    // Count query
    countQuery := fmt.Sprintf("SELECT COUNT(*) FROM mdm_devices WHERE %s", whereClause)
    var total int64
    if err := r.DB.Get(&total, countQuery, args...); err != nil {
        return nil, 0, err
    }

    // Data query
    limit := filter.Limit
    if limit <= 0 {
        limit = 50
    }
    offset := filter.Offset

    query := fmt.Sprintf(`
        SELECT * FROM mdm_devices
        WHERE %s
        ORDER BY updated_at DESC
        LIMIT ? OFFSET ?`, whereClause)

    args = append(args, limit, offset)

    var devices []*models.MDMDevice
    if err := r.DB.Select(&devices, query, args...); err != nil {
        return nil, 0, err
    }

    return devices, total, nil
}

// ... 其他方法实现类似
```

##### 1.1.4 Service 实现

**文件**: `internal/services/device_service.go` (NEW)

```go
package services

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/pseudolution/freekiosk-hub/internal/models"
    "github.com/pseudolution/freekiosk-hub/internal/repositories"
)

// DeviceService 设备服务接口
type DeviceService interface {
    // 设备管理
    RegisterDevice(ctx context.Context, tenantID string, req *RegisterDeviceRequest) (*models.MDMDevice, error)
    GetDevice(ctx context.Context, deviceID string) (*models.MDMDevice, error)
    UpdateDevice(ctx context.Context, deviceID string, req *UpdateDeviceRequest) error
    DeleteDevice(ctx context.Context, deviceID string) error
    ListDevices(ctx context.Context, tenantID string, filter *models.DeviceSearchFilter) ([]*models.MDMDevice, int64, error)

    // 设备控制
    LockDevice(ctx context.Context, deviceID string) error
    RebootDevice(ctx context.Context, deviceID string) error
    FactoryResetDevice(ctx context.Context, deviceID string) error
    GetDeviceLocation(ctx context.Context, deviceID string) (*GPSLocation, error)

    // 分组管理
    CreateGroup(ctx context.Context, tenantID string, req *CreateGroupRequest) (*models.DeviceGroup, error)
    ListGroups(ctx context.Context, tenantID string) ([]*models.DeviceGroup, error)
    AssignDevicesToGroup(ctx context.Context, deviceIDs []string, groupID string) error

    // 事件追踪
    RecordDeviceEvent(ctx context.Context, deviceID, eventType string, data interface{}) error
    GetDeviceEvents(ctx context.Context, deviceID string, limit int) ([]*models.DeviceEvent, error)
}

type DeviceServiceImpl struct {
    deviceRepo repositories.MDMTabletRepository
    groupRepo repositories.MDMTabletRepository // 复用同一 repo
    mqttSvc   *MQTTService
    cmdSvc    CommandService
}

func NewDeviceService(
    deviceRepo repositories.MDMTabletRepository,
    groupRepo repositories.MDMTabletRepository,
    mqttSvc *MQTTService,
    cmdSvc CommandService,
) *DeviceServiceImpl {
    return &DeviceServiceImpl{
        deviceRepo: deviceRepo,
        groupRepo: groupRepo,
        mqttSvc: mqttSvc,
        cmdSvc: cmdSvc,
    }
}

type RegisterDeviceRequest struct {
    Number        string            `json:"number"`
    Name          string            `json:"name"`
    Description   string            `json:"description,omitempty"`
    IMEI          string            `json:"imei,omitempty"`
    Phone         string            `json:"phone,omitempty"`
    Model         string            `json:"model,omitempty"`
    Manufacturer  string            `json:"manufacturer,omitempty"`
    OSVersion     string            `json:"os_version,omitempty"`
    SDKVersion    int               `json:"sdk_version,omitempty"`
    AppVersion    string            `json:"app_version,omitempty"`
    AppVersionCode int              `json:"app_version_code,omitempty"`
    Carrier      string            `json:"carrier,omitempty"`
    Metadata      map[string]string `json:"metadata,omitempty"`
}

type UpdateDeviceRequest struct {
    Name            *string `json:"name,omitempty"`
    Description     *string `json:"description,omitempty"`
    Status          *string `json:"status,omitempty"`
    ConfigurationID *string `json:"configuration_id,omitempty"`
    GroupID         *string `json:"group_id,omitempty"`
    Metadata        map[string]string `json:"metadata,omitempty"`
}

type GPSLocation struct {
    Lat      float64 `json:"lat"`
    Lng      float64 `json:"lng"`
    Timestamp int64   `json:"timestamp"`
}

func (s *DeviceServiceImpl) RegisterDevice(ctx context.Context, tenantID string, req *RegisterDeviceRequest) (*models.MDMDevice, error) {
    device := &models.MDMDevice{
        Number:        req.Number,
        Name:          req.Name,
        Description:   req.Description,
        IMEI:          req.IMEI,
        Phone:         req.Phone,
        Model:         req.Model,
        Manufacturer:  req.Manufacturer,
        OSVersion:     req.OSVersion,
        SDKVersion:    req.SDKVersion,
        AppVersion:    req.AppVersion,
        AppVersionCode: req.AppVersionCode,
        Carrier:       req.Carrier,
        Status:        string(models.DeviceStatusActive),
        TenantID:      tenantID,
    }

    if req.Metadata != nil {
        metadataJSON, _ := json.Marshal(req.Metadata)
        device.Metadata = string(metadataJSON)
    }

    if err := s.deviceRepo.CreateDevice(device); err != nil {
        return nil, fmt.Errorf("failed to create device: %w", err)
    }

    // 记录注册事件
    s.RecordDeviceEvent(ctx, device.ID, "device_registered", map[string]string{
        "number": device.Number,
    })

    return device, nil
}

func (s *DeviceServiceImpl) LockDevice(ctx context.Context, deviceID string) error {
    device, err := s.deviceRepo.GetDeviceByID(deviceID)
    if err != nil {
        return fmt.Errorf("device not found: %w", err)
    }

    // 通过 MQTT 发送锁定命令
    cmd := &models.Command{
        Type:   models.CommandLock,
        Params: json.RawMessage(`{}`),
    }

    result, err := s.cmdSvc.SendCommand(ctx, device.TenantID, deviceID, cmd)
    if err != nil {
        return fmt.Errorf("failed to send lock command: %w", err)
    }

    if !result.Success {
        return fmt.Errorf("lock command failed: %s", result.Error)
    }

    // 记录事件
    s.RecordDeviceEvent(ctx, deviceID, "device_locked", nil)

    return nil
}

func (s *DeviceServiceImpl) GetDeviceLocation(ctx context.Context, deviceID string) (*GPSLocation, error) {
    location, err := s.deviceRepo.GetDeviceLocation(deviceID)
    if err != nil {
        return nil, err
    }

    return &GPSLocation{
        Lat:      location.Lat,
        Lng:      location.Lng,
        Timestamp: location.Timestamp,
    }, nil
}
```

##### 1.1.5 API Handler 实现

**文件**: `internal/api/mdm_device_handler.go` (NEW)

```go
package api

import (
    "net/http"
    "strconv"

    "github.com/labstack/echo/v4"
    "github.com/pseudolution/freekiosk-hub/internal/models"
    "github.com/pseudolution/freekiosk-hub/internal/services"
)

// MDMDeviceHandler MDM设备管理处理器
type MDMDeviceHandler struct {
    deviceSvc services.DeviceService
}

func NewMDMDeviceHandler(deviceSvc services.DeviceService) *MDMDeviceHandler {
    return &MDMDeviceHandler{deviceSvc: deviceSvc}
}

// RegisterRoutes 注册 MDM 设备路由
func (h *MDMDeviceHandler) RegisterRoutes(g *echo.Group, authMiddleware echo.MiddlewareFunc) {
    devices := g.Group("/devices", authMiddleware)
    {
        devices.GET("", h.ListDevices)
        devices.POST("", h.CreateDevice)
        devices.GET("/:id", h.GetDevice)
        devices.PUT("/:id", h.UpdateDevice)
        devices.DELETE("/:id", h.DeleteDevice)
        devices.POST("/search", h.SearchDevices)

        // 设备控制
        devices.POST("/:id/lock", h.LockDevice)
        devices.POST("/:id/reboot", h.RebootDevice)
        devices.POST("/:id/factory-reset", h.FactoryReset)
        devices.POST("/:id/location", h.GetLocation)

        // 分组
        devices.POST("/:id/group", h.AssignGroup)
    }

    // 分组路由
    groups := g.Group("/groups", authMiddleware)
    {
        groups.GET("", h.ListGroups)
        groups.POST("", h.CreateGroup)
        groups.PUT("/:id", h.UpdateGroup)
        groups.DELETE("/:id", h.DeleteGroup)
    }
}

// ListDevices GET /api/v2/devices
func (h *MDMDeviceHandler) ListDevices(c echo.Context) error {
    tenantID := c.Get("tenant_id").(string)

    limit, _ := strconv.Atoi(c.QueryParam("limit"))
    offset, _ := strconv.Atoi(c.QueryParam("offset"))

    filter := &models.DeviceSearchFilter{
        TenantID: tenantID,
        Limit:    limit,
        Offset:   offset,
    }

    devices, total, err := h.deviceSvc.ListDevices(c.Request().Context(), tenantID, filter)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "devices": devices,
        "total":   total,
        "limit":   limit,
        "offset":  offset,
    })
}

// CreateDevice POST /api/v2/devices
func (h *MDMDeviceHandler) CreateDevice(c echo.Context) error {
    tenantID := c.Get("tenant_id").(string)

    var req services.RegisterDeviceRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }

    device, err := h.deviceSvc.RegisterDevice(c.Request().Context(), tenantID, &req)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }

    return c.JSON(http.StatusCreated, device)
}

// GetDevice GET /api/v2/devices/:id
func (h *MDMDeviceHandler) GetDevice(c echo.Context) error {
    deviceID := c.Param("id")

    device, err := h.deviceSvc.GetDevice(c.Request().Context(), deviceID)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, "device not found")
    }

    return c.JSON(http.StatusOK, device)
}

// SearchDevices POST /api/v2/devices/search
func (h *MDMDeviceHandler) SearchDevices(c echo.Context) error {
    tenantID := c.Get("tenant_id").(string)

    var filter models.DeviceSearchFilter
    if err := c.Bind(&filter); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }
    filter.TenantID = tenantID

    devices, total, err := h.deviceSvc.ListDevices(c.Request().Context(), tenantID, &filter)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
        "devices": devices,
        "total":   total,
    })
}

// LockDevice POST /api/v2/devices/:id/lock
func (h *MDMDeviceHandler) LockDevice(c echo.Context) error {
    deviceID := c.Param("id")

    if err := h.deviceSvc.LockDevice(c.Request().Context(), deviceID); err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }

    return c.JSON(http.StatusOK, map[string]string{
        "message": "Lock command sent",
    })
}
```

##### 1.1.6 路由注册

**文件**: `internal/api/router.go` (修改)

```go
// 在 setupRoutes() 中添加
if s.DeviceSvc != nil {
    mdmH := NewMDMDeviceHandler(s.DeviceSvc)
    mdmRoutes := s.Echo.Group("/api/v2/mdm")
    mdmH.RegisterRoutes(mdmRoutes, AuthMiddleware)
}
```

---

#### 第 3-4 周：设备注册和同步

##### 1.2.1 设备信息同步协议

**Android 端 - KioskHttpServer.kt 扩展**

```kotlin
// 新增端点
@POST("/api/mdm/register")
fun registerDevice(@Body request: Map<String, Any>): Response {
    val deviceInfo = buildDeviceInfo()
    return sendToHub("/api/v2/mdm/devices", deviceInfo)
}

@POST("/api/mdm/heartbeat")
fun sendHeartbeat(@Body status: Map<String, Any>): Response {
    return sendToHub("/api/v2/mdm/devices/${deviceId}/heartbeat", status)
}

@POST("/api/mdm/location")
fun reportLocation(@Body location: Map<String, Any>): Response {
    return sendToHub("/api/v2/mdm/devices/${deviceId}/location", location)
}

private fun buildDeviceInfo(): Map<String, Any> {
    val pm = getPackageManager()
    val packageInfo = pm.getPackageInfo(packageName, 0)

    return mapOf(
        "number" to getDeviceNumber(),      // 设备唯一标识
        "name" to getDeviceName(),           // 设备名称
        "imei" to getIMEI(),                // IMEI
        "phone" to getPhoneNumber(),        // 手机号
        "model" to Build.MODEL,             // 机型
        "manufacturer" to Build.MANUFACTURER,// 厂商
        "os_version" to Build.VERSION.RELEASE, // Android 版本
        "sdk_version" to Build.VERSION.SDK_INT, // SDK 版本
        "app_version" to packageInfo.versionName,
        "app_version_code" to packageInfo.versionCode,
        "carrier" to getCarrierName()       // 运营商
    )
}
```

##### 1.2.2 Hub 端心跳处理

```go
// internal/services/device_service.go 新增方法

type HeartbeatRequest struct {
    BatteryLevel int    `json:"battery_level"`
    BatteryStatus string `json:"battery_status"`
    ScreenOn      bool   `json:"screen_on"`
    MemoryUsage   int    `json:"memory_usage"`
    StorageUsage  int    `json:"storage_usage"`
    ActiveApp    string `json:"active_app,omitempty"`
    Timestamp    int64  `json:"timestamp"`
}

func (s *DeviceServiceImpl) ProcessHeartbeat(ctx context.Context, deviceID string, req *HeartbeatRequest) error {
    device, err := s.deviceRepo.GetDeviceByID(deviceID)
    if err != nil {
        return err
    }

    // 更新最后在线时间
    now := time.Now().Unix()
    device.LastSeen = &now

    // 记录电池状态到事件
    eventData, _ := json.Marshal(map[string]interface{}{
        "battery_level": req.BatteryLevel,
        "battery_status": req.BatteryStatus,
        "screen_on": req.ScreenOn,
        "memory_usage": req.MemoryUsage,
        "timestamp": req.Timestamp,
    })

    event := &models.DeviceEvent{
        ID:        generateUUID(),
        DeviceID:  deviceID,
        EventType: "heartbeat",
        EventData: string(eventData),
        CreatedAt: now,
    }

    s.deviceRepo.RecordEvent(event)
    s.deviceRepo.UpdateDevice(device)

    return nil
}
```

---

#### 第 5-6 周：远程命令扩展

##### 1.3.1 命令类型扩展

**文件**: `internal/models/command.go` (扩展)

```go
// 新增命令类型
const (
    // 设备控制
    CommandLock         = "lock"
    CommandReboot       = "reboot"
    CommandFactoryReset  = "factoryReset"
    CommandScreenshot    = "screenshot"
    CommandRing         = "ring"           // 设备响铃
    CommandLocation     = "location"       // 获取位置
    CommandTimeSync     = "timeSync"       // 时间同步

    // 远程控制
    CommandInput        = "input"          // 远程输入
    CommandClick        = "click"          // 点击
    CommandSwipe        = "swipe"          // 滑动

    // 应用管理
    CommandInstallApp   = "installApp"
    CommandUninstallApp = "uninstallApp"
    CommandUpdateApp   = "updateApp"

    // 配置管理
    CommandApplyConfig  = "applyConfig"
    CommandClearConfig = "clearConfig"
)

// ScreenshotResult 截图结果
type ScreenshotResult struct {
    ImageBase64 string `json:"image_base64"`
    Width       int    `json:"width"`
    Height      int    `json:"height"`
    Timestamp   int64  `json:"timestamp"`
}

// LocationResult 位置结果
type LocationResult struct {
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
    Accuracy  float64 `json:"accuracy"`
    Altitude  float64 `json:"altitude"`
    Timestamp int64   `json:"timestamp"`
}
```

##### 1.3.2 MQTT 命令分发

**文件**: `internal/services/mqtt_handler.go` (NEW)

```go
package services

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"

    "github.com/pseudolution/freekiosk-hub/internal/models"
    "github.com/pseudolution/freekiosk-hub/internal/repositories"
)

// MQTTCommandHandler MQTT命令处理器
type MQTTCommandHandler struct {
    deviceRepo repositories.MDMTabletRepository
    cmdSvc    CommandService
}

func NewMQTTCommandHandler(
    deviceRepo repositories.MDMTabletRepository,
    cmdSvc CommandService,
) *MQTTCommandHandler {
    return &MQTTCommandHandler{
        deviceRepo: deviceRepo,
        cmdSvc:    cmdSvc,
    }
}

// HandleDeviceCommand 处理设备命令
// topic: kiosk/{tenantId}/{deviceId}/command
func (h *MQTTCommandHandler) HandleDeviceCommand(ctx context.Context, topic string, payload []byte) error {
    // 解析 topic
    // kiosk/{tenantId}/{deviceId}/command
    parts := splitTopic(topic)
    if len(parts) < 4 {
        return fmt.Errorf("invalid topic format: %s", topic)
    }
    tenantID := parts[1]
    deviceID := parts[2]

    var cmd models.Command
    if err := json.Unmarshal(payload, &cmd); err != nil {
        return fmt.Errorf("failed to unmarshal command: %w", err)
    }

    // 记录命令
    slog.Info("Received command",
        "device_id", deviceID,
        "type", cmd.Type,
        "command_id", cmd.ID)

    // 根据命令类型处理
    switch cmd.Type {
    case models.CommandScreenshot:
        return h.handleScreenshot(ctx, tenantID, deviceID, &cmd)
    case models.CommandLocation:
        return h.handleLocation(ctx, tenantID, deviceID, &cmd)
    case models.CommandRing:
        return h.handleRing(ctx, tenantID, deviceID, &cmd)
    case models.CommandInstallApp:
        return h.handleInstallApp(ctx, tenantID, deviceID, &cmd)
    case models.CommandUninstallApp:
        return h.handleUninstallApp(ctx, tenantID, deviceID, &cmd)
    default:
        // 通用命令走 CommandService
        result, err := h.cmdSvc.SendCommand(ctx, tenantID, deviceID, &cmd)
        if err != nil {
            return fmt.Errorf("command failed: %w", err)
        }
        if !result.Success {
            return fmt.Errorf("command error: %s", result.Error)
        }
        return nil
    }
}

func (h *MQTTCommandHandler) handleScreenshot(ctx context.Context, tenantID, deviceID string, cmd *models.Command) error {
    // 发送截图命令到设备
    result, err := h.cmdSvc.SendCommand(ctx, tenantID, deviceID, cmd)
    if err != nil {
        return err
    }

    // 截图结果通过 REST 上报
    // 设备 POST /api/v2/mdm/devices/{id}/screenshot
    slog.Info("Screenshot captured", "device_id", deviceID, "success", result.Success)
    return nil
}

func (h *MQTTCommandHandler) handleLocation(ctx context.Context, tenantID, deviceID string, cmd *models.Command) error {
    // 发送立即定位命令
    result, err := h.cmdSvc.SendCommand(ctx, tenantID, deviceID, cmd)
    if err != nil {
        return err
    }

    // 位置结果通过 REST 上报
    // 设备 POST /api/v2/mdm/devices/{id}/location
    slog.Info("Location requested", "device_id", deviceID)
    return nil
}
```

---

#### 第 7-8 周：应用管理 MVP

##### 1.4.1 应用管理 Model

**文件**: `internal/models/application.go` (NEW)

```go
package models

import "time"

// Application 应用模型
type Application struct {
    ID                string     `json:"id" db:"id"`
    Name              string     `json:"name" db:"name"`
    PackageName       string     `json:"package_name" db:"package_name"` // APK 包名
    Description       string     `json:"description" db:"description"`
    Category          string     `json:"category" db:"category"`           // education, utility, etc.
    IconPath          string     `json:"icon_path" db:"icon_path"`
    APKPath           string     `json:"apk_path" db:"apk_path"`
    Version           string     `json:"version" db:"version"`
    VersionCode       int        `json:"version_code" db:"version_code"`
    FileSize          int64      `json:"file_size" db:"file_size"`
    Permissions       string     `json:"permissions" db:"permissions"`     // JSON array
    MinSDKVersion     int        `json:"min_sdk_version" db:"min_sdk_version"`
    InstallType       string     `json:"install_type" db:"install_type"`   // kiosk, required, optional
    TenantID          string     `json:"tenant_id" db:"tenant_id"`
    CreatedAt         int64      `json:"created_at" db:"created_at"`
    UpdatedAt         int64      `json:"updated_at" db:"updated_at"`
}

// ApplicationInstall 应用安装记录
type ApplicationInstall struct {
    ID           string    `json:"id" db:"id"`
    ApplicationID string    `json:"application_id" db:"application_id"`
    DeviceID     string    `json:"device_id" db:"device_id"`
    Version      string    `json:"version" db:"version"`
    Status       string    `json:"status" db:"status"` // pending, installing, installed, failed
    ErrorMessage string    `json:"error_message" db:"error_message"`
    InstalledAt  *int64    `json:"installed_at" db:"installed_at"`
    CreatedAt    int64     `json:"created_at" db:"created_at"`
}

// AppInstallType 应用安装类型
const (
    AppInstallTypeKiosk   = "kiosk"    // Kiosk 专用应用
    AppInstallTypeRequired = "required" // 必装应用
    AppInstallTypeOptional = "optional"  // 可选应用
)
```

##### 1.4.2 应用存储服务

**文件**: `internal/services/app_storage_service.go` (NEW)

```go
package services

import (
    "fmt"
    "io"
    "mime/multipart"
    "os"
    "path/filepath"
    "strings"

    "github.com/google/uuid"
    "github.com/pseudolution/freekiosk-hub/internal/models"
)

// AppStorageService 应用存储服务
type AppStorageService interface {
    SaveAPK(tenantID string, file *multipart.FileHeader) (*models.Application, error)
    GetAPKPath(appID string) (string, error)
    DeleteAPK(appID string) error
    GenerateIcon(appID string) error
}

type LocalAppStorageService struct {
    basePath string
}

func NewLocalAppStorageService(basePath string) *LocalAppStorageService {
    // 确保目录存在
    os.MkdirAll(filepath.Join(basePath, "apk"), 0755)
    os.MkdirAll(filepath.Join(basePath, "icons"), 0755)
    return &LocalAppStorageService{basePath: basePath}
}

func (s *LocalAppStorageService) SaveAPK(tenantID string, file *multipart.FileHeader) (*models.Application, error) {
    // 验证文件类型
    if !strings.HasSuffix(file.Filename, ".apk") {
        return nil, fmt.Errorf("only APK files are allowed")
    }

    // 生成唯一 ID
    appID := uuid.New().String()

    // 保存文件
    apkDir := filepath.Join(s.basePath, "apk", tenantID)
    os.MkdirAll(apkDir, 0755)

    apkPath := filepath.Join(apkDir, fmt.Sprintf("%s.apk", appID))

    src, err := file.Open()
    if err != nil {
        return nil, fmt.Errorf("failed to open uploaded file: %w", err)
    }
    defer src.Close()

    dst, err := os.Create(apkPath)
    if err != nil {
        return nil, fmt.Errorf("failed to create destination file: %w", err)
    }
    defer dst.Close()

    if _, err := io.Copy(dst, src); err != nil {
        return nil, fmt.Errorf("failed to save APK: %w", err)
    }

    // 解析 APK 信息（使用 aapt 或其他工具）
    // 这里简化处理，实际需要调用 Android SDK 的 aapt 或 apktool
    app := &models.Application{
        ID:       appID,
        APKPath:  apkPath,
        FileSize: file.Size,
    }

    return app, nil
}

func (s *LocalAppStorageService) GetAPKPath(appID string) (string, error) {
    // 需要从数据库获取实际路径，这里简化
    path := filepath.Join(s.basePath, "apk", fmt.Sprintf("%s.apk", appID))
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return "", fmt.Errorf("APK not found: %s", appID)
    }
    return path, nil
}
```

---

### Android 端实现方案

#### 第 1-2 周：设备注册模块

##### 1.1 Android 端设备信息收集

**文件**: `android/app/src/main/java/com/freekiosk/DeviceInfoModule.kt` (NEW)

```kotlin
package com.freekiosk

import android.app.ActivityManager
import android.content.Context
import android.content.pm.PackageManager
import android.net.ConnectivityManager
import android.net.wifi.WifiManager
import android.os.BatteryManager
import android.os.Build
import android.provider.Settings
import com.facebook.react.bridge.*
import java.io.BufferedReader
import java.io.FileReader
import java.util.UUID

class DeviceInfoModule(reactContext: ReactApplicationContext) : ReactContextBaseJavaModule(reactContext) {

    override fun getName() = "DeviceInfoModule"

    // 获取设备唯一标识
    @ReactMethod
    fun getDeviceId(promise: Promise) {
        try {
            val androidId = Settings.Secure.getString(
                reactApplicationContext.contentResolver,
                Settings.Secure.ANDROID_ID
            )
            // 使用 Android ID + UUID 作为设备唯一标识
            val deviceId = if (androidId != null && androidId != "9774d56d682e549c") {
                androidId
            } else {
                UUID.randomUUID().toString()
            }
            promise.resolve(deviceId)
        } catch (e: Exception) {
            promise.reject("DEVICE_ID_ERROR", e.message)
        }
    }

    // 获取设备信息 Map
    @ReactMethod
    fun getDeviceInfo(promise: Promise) {
        try {
            val pm = reactApplicationContext.packageManager
            val packageInfo = pm.getPackageInfo(reactApplicationContext.packageName, 0)

            val params = Arguments.createMap().apply {
                putString("number", getDeviceIdSync())
                putString("name", getDeviceName())
                putString("imei", getIMEI())
                putString("phone", getPhoneNumber())
                putString("model", Build.MODEL)
                putString("manufacturer", Build.MANUFACTURER)
                putString("os_version", Build.VERSION.RELEASE)
                putInt("sdk_version", Build.VERSION.SDK_INT)
                putString("app_version", packageInfo.versionName ?: "1.0")
                putInt("app_version_code", packageInfo.versionCode)
                putString("carrier", getCarrierName())
            }
            promise.resolve(params)
        } catch (e: Exception) {
            promise.reject("DEVICE_INFO_ERROR", e.message)
        }
    }

    private fun getDeviceIdSync(): String {
        return Settings.Secure.getString(
            reactApplicationContext.contentResolver,
            Settings.Secure.ANDROID_ID
        ) ?: "unknown"
    }

    private fun getDeviceName(): String {
        return "${Build.MANUFACTURER} ${Build.MODEL}"
    }

    @Suppress("DEPRECATION")
    private fun getIMEI(): String {
        return try {
            val telephony = reactApplicationContext.getSystemService(Context.TELEPHONY_SERVICE) as android.telephony.TelephonyManager
            telephony.deviceId ?: "unknown"
        } catch (e: SecurityException) {
            "unknown"
        }
    }

    @Suppress("DEPRECATION")
    private fun getPhoneNumber(): String {
        return try {
            val telephony = reactApplicationContext.getSystemService(Context.TELEPHONY_SERVICE) as android.telephony.TelephonyManager
            telephony.line1Number ?: "unknown"
        } catch (e: SecurityException) {
            "unknown"
        }
    }

    @Suppress("DEPRECATION")
    private fun getCarrierName(): String {
        return try {
            val telephony = reactApplicationContext.getSystemService(Context.TELEPHONY_SERVICE) as android.telephony.TelephonyManager
            telephony.networkOperatorName ?: "unknown"
        } catch (e: Exception) {
            "unknown"
        }
    }

    // 电池状态
    @ReactMethod
    fun getBatteryStatus(promise: Promise) {
        val batteryManager = reactApplicationContext.getSystemService(Context.BATTERY_SERVICE) as BatteryManager
        val level = batteryManager.getIntProperty(BatteryManager.BATTERY_PROPERTY_CAPACITY)
        val status = batteryManager.isCharging

        val params = Arguments.createMap().apply {
            putInt("level", level)
            putBoolean("is_charging", status)
            putString("status", if (status) "charging" else "discharging")
        }
        promise.resolve(params)
    }

    // 系统内存
    @ReactMethod
    fun getMemoryInfo(promise: Promise) {
        val activityManager = reactApplicationContext.getSystemService(Context.ACTIVITY_SERVICE) as ActivityManager
        val memInfo = ActivityManager.MemoryInfo()
        activityManager.getMemoryInfo(memInfo)

        val params = Arguments.createMap().apply {
            putDouble("total_memory", memInfo.totalMem.toDouble())
            putDouble("available_memory", memInfo.availMem.toDouble())
            putBoolean("low_memory", memInfo.lowMemory)
        }
        promise.resolve(params)
    }
}
```

##### 1.2 注册流程

**文件**: `android/app/src/main/java/com/freekiosk/MDMRegistration.kt` (NEW)

```kotlin
package com.freekiosk

import android.util.Log
import com.facebook.react.bridge.*
import kotlinx.coroutines.*
import java.io.BufferedReader
import java.io.InputStreamReader
import java.net.HttpURLConnection
import java.net.URL

class MDMRegistration(private val context: Context) {

    private val hubUrl: String = getHubUrl()
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())

    companion object {
        private const val TAG = "MDMRegistration"
    }

    // 注册设备到 Hub
    fun registerDevice(deviceInfo: WritableMap, callback: Callback) {
        scope.launch {
            try {
                val response = postToHub("/api/v2/mdm/devices", deviceInfo)
                withContext(Dispatchers.Main) {
                    callback(null, response)
                }
            } catch (e: Exception) {
                Log.e(TAG, "Registration failed", e)
                withContext(Dispatchers.Main) {
                    callback(e, null)
                }
            }
        }
    }

    // 发送心跳
    fun sendHeartbeat(status: WritableMap, callback: Callback) {
        scope.launch {
            try {
                val deviceId = getDeviceId()
                val response = postToHub("/api/v2/mdm/devices/$deviceId/heartbeat", status)
                withContext(Dispatchers.Main) {
                    callback(null, response)
                }
            } catch (e: Exception) {
                Log.e(TAG, "Heartbeat failed", e)
                withContext(Dispatchers.Main) {
                    callback(e, null)
                }
            }
        }
    }

    // 上报位置
    fun reportLocation(lat: Double, lng: Double, callback: Callback) {
        scope.launch {
            try {
                val deviceId = getDeviceId()
                val locationData = Arguments.createMap().apply {
                    putDouble("lat", lat)
                    putDouble("lng", lng)
                    putDouble("timestamp", System.currentTimeMillis())
                }
                val response = postToHub("/api/v2/mdm/devices/$deviceId/location", locationData)
                withContext(Dispatchers.Main) {
                    callback(null, response)
                }
            } catch (e: Exception) {
                Log.e(TAG, "Location report failed", e)
                withContext(Dispatchers.Main) {
                    callback(e, null)
                }
            }
        }
    }

    private suspend fun postToHub(endpoint: String, data: WritableMap): WritableMap {
        return withContext(Dispatchers.IO) {
            val url = URL("$hubUrl$endpoint")
            val connection = url.openConnection() as HttpURLConnection
            connection.requestMethod = "POST"
            connection.setRequestProperty("Content-Type", "application/json")
            connection.doOutput = true

            // 写入 body
            connection.outputStream.use { os ->
                val body = convertMapToJSON(data)
                os.write(body.toByteArray())
            }

            val responseCode = connection.responseCode
            if (responseCode !in 200..299) {
                throw Exception("HTTP $responseCode")
            }

            // 读取响应
            val reader = BufferedReader(InputStreamReader(connection.inputStream))
            val response = reader.readText()
            reader.close()

            // 解析 JSON 响应
            convertJSONToMap(response)
        }
    }

    private fun getHubUrl(): String {
        // 从 SharedPreferences 读取 Hub URL
        val prefs = context.getSharedPreferences("freekiosk", Context.MODE_PRIVATE)
        return prefs.getString("hub_url", "http://localhost:8081") ?: "http://localhost:8081"
    }

    private fun getDeviceId(): String {
        val prefs = context.getSharedPreferences("freekiosk", Context.MODE_PRIVATE)
        return prefs.getString("device_id", "unknown") ?: "unknown"
    }

    private fun convertMapToJSON(map: WritableMap): String {
        // 简化实现，实际需要使用 JSON 库
        val sb = StringBuilder()
        sb.append("{")
        val keys = map.keys()
        var first = true
        while (keys.hasNext()) {
            if (!first) sb.append(",")
            first = false
            val key = keys.next()
            sb.append("\"$key\":")
            when (val value = map.getDynamic(key)) {
                is String -> sb.append("\"${value.asString()}\"")
                is Double -> sb.append(value.asDouble())
                is Int -> sb.append(value.asInt())
                is Boolean -> sb.append(value)
                else -> sb.append("null")
            }
        }
        sb.append("}")
        return sb.toString()
    }

    private fun convertJSONToMap(json: String): WritableMap {
        // 简化实现，实际需要使用 JSON 解析库
        return Arguments.createMap()
    }
}
```

---

#### 第 3-4 周：命令执行扩展

##### 2.1 命令处理器扩展

**文件**: `android/app/src/main/java/com/freekiosk/command/MDMCommandHandler.kt` (NEW)

```kotlin
package com.freekiosk.command

import android.app.admin.DeviceAdminInfo
import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.media.projection.MediaProjectionManager
import android.os.Build
import android.os.PowerManager
import android.os.SystemClock
import android.util.Base64
import android.view.InputDevice
import android.view.KeyEvent
import com.facebook.react.bridge.*
import kotlinx.coroutines.*
import org.json.JSONObject
import java.io.ByteArrayOutputStream
import java.io.DataOutputStream
import java.net.HttpURLConnection
import java.net.URL

class MDMCommandHandler(private val context: Context) {

    private val scope = CoroutineScope(Dispatchers.Main + SupervisorJob())
    private val deviceAdmin = ComponentName(context, FreeKioskDeviceAdminReceiver::class.java)

    // 命令类型
    object CommandType {
        const val LOCK = "lock"
        const val REBOOT = "reboot"
        const val FACTORY_RESET = "factoryReset"
        const val SCREENSHOT = "screenshot"
        const val LOCATION = "location"
        const val RING = "ring"
        const val INPUT = "input"
        const val INSTALL_APP = "installApp"
        const val UNINSTALL_APP = "uninstallApp"
    }

    // 执行命令
    fun executeCommand(command: JSONObject, callback: (Result) -> Unit) {
        val type = command.optString("type")
        val params = command.optJSONObject("params") ?: JSONObject()
        val commandId = command.optString("id")

        when (type) {
            CommandType.LOCK -> handleLock(commandId, callback)
            CommandType.REBOOT -> handleReboot(commandId, callback)
            CommandType.FACTORY_RESET -> handleFactoryReset(commandId, callback)
            CommandType.SCREENSHOT -> handleScreenshot(commandId, callback)
            CommandType.LOCATION -> handleLocation(commandId, callback)
            CommandType.RING -> handleRing(commandId, callback)
            CommandType.INPUT -> handleInput(commandId, params, callback)
            else -> callback(Result.failure("Unknown command: $type"))
        }
    }

    // 锁定设备
    private fun handleLock(commandId: String, callback: (Result) -> Unit) {
        try {
            val powerManager = context.getSystemService(Context.POWER_SERVICE) as PowerManager
            val lockFlags = android.view.WindowManager.LayoutParams.FLAG_BLOCK_SCREEN_CAPTURE or
                           android.view.WindowManager.LayoutParams.FLAG_SECURE
            // 实际锁定需要启动锁屏 Activity 或使用 Device Owner API
            callback(Result.success(JSONObject().put("message", "Device locked")))
        } catch (e: Exception) {
            callback(Result.failure(e.message ?: "Lock failed"))
        }
    }

    // 重启设备
    private fun handleReboot(commandId: String, callback: (Result) -> Unit) {
        try {
            val intent = Intent(Intent.ACTION_REBOOT)
            intent.putExtra("nowait", 1)
            intent.putExtra("force", 1)
            context.sendBroadcast(intent)
            callback(Result.success(JSONObject().put("message", "Reboot initiated")))
        } catch (e: Exception) {
            callback(Result.failure(e.message ?: "Reboot failed"))
        }
    }

    // 恢复出厂设置
    private fun handleFactoryReset(commandId: String, callback: (Result) -> Unit) {
        try {
            // 需要 Device Owner 权限
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
                val intent = Intent(android.admin.DeviceAdminReceiver.ACTION_DEVICE_ADMIN_ENABLED)
                // 实际需要调用 DevicePolicyManager.wipeData()
            }
            callback(Result.success(JSONObject().put("message", "Factory reset initiated")))
        } catch (e: Exception) {
            callback(Result.failure(e.message ?: "Factory reset failed"))
        }
    }

    // 截图
    private fun handleScreenshot(commandId: String, callback: (Result) -> Unit) {
        scope.launch {
            try {
                // 使用 MediaProjection API 截图
                val bitmap = captureScreen()
                if (bitmap != null) {
                    val outputStream = ByteArrayOutputStream()
                    bitmap.compress(Bitmap.CompressFormat.PNG, 100, outputStream)
                    val base64 = Base64.encodeToString(outputStream.toByteArray(), Base64.DEFAULT)

                    val result = JSONObject().apply {
                        put("image_base64", base64)
                        put("width", bitmap.width)
                        put("height", bitmap.height)
                        put("timestamp", System.currentTimeMillis())
                    }
                    callback(Result.success(result))
                } else {
                    callback(Result.failure("Screenshot failed"))
                }
            } catch (e: Exception) {
                callback(Result.failure(e.message ?: "Screenshot failed"))
            }
        }
    }

    // 获取位置
    private fun handleLocation(commandId: String, callback: (Result) -> Unit) {
        scope.launch {
            try {
                // 实际需要使用 FusedLocationProvider 或 GPS
                val result = JSONObject().apply {
                    put("latitude", 39.9042)  // 示例数据
                    put("longitude", 116.4074)
                    put("accuracy", 10.0)
                    put("altitude", 0.0)
                    put("timestamp", System.currentTimeMillis())
                }
                callback(Result.success(result))
            } catch (e: Exception) {
                callback(Result.failure(e.message ?: "Location failed"))
            }
        }
    }

    // 设备响铃
    private fun handleRing(commandId: String, callback: (Result) -> Unit) {
        try {
            // 播放提示音 + 震动
            val mediaPlayer = android.media.MediaPlayer.create(context, android.provider.Settings.System.DEFAULT_RINGTONE_URI)
            mediaPlayer?.apply {
                isLooping = true
                start()
            }

            val vibrator = context.getSystemService(Context.VIBRATOR_SERVICE) as android.os.Vibrator
            vibrator.vibrate(longArrayOf(0, 500, 200, 500), -1)

            // 5秒后停止
            scope.launch {
                delay(5000)
                mediaPlayer?.stop()
                mediaPlayer?.release()
                vibrator.cancel()
            }

            callback(Result.success(JSONObject().put("message", "Device ringing")))
        } catch (e: Exception) {
            callback(Result.failure(e.message ?: "Ring failed"))
        }
    }

    // 远程输入
    private fun handleInput(commandId: String, params: JSONObject, callback: (Result) -> Unit) {
        try {
            val inputType = params.optString("type")
            val x = params.optDouble("x", -1.0)
            val y = params.optDouble("y", -1.0)
            val keyCode = params.optInt("keyCode", -1)

            when (inputType) {
                "tap" -> injectTap(x.toInt(), y.toInt())
                "swipe" -> {
                    val x2 = params.optDouble("x2", x)
                    val y2 = params.optDouble("y2", y)
                    injectSwipe(x.toInt(), y.toInt(), x2.toInt(), y2.toInt())
                }
                "key" -> injectKey(keyCode)
            }

            callback(Result.success(JSONObject().put("message", "Input injected")))
        } catch (e: Exception) {
            callback(Result.failure(e.message ?: "Input failed"))
        }
    }

    private fun injectTap(x: Int, y: Int) {
        // 需要 accessibility service 或 root
        // 这里简化处理
    }

    private fun injectSwipe(x1: Int, y1: Int, x2: Int, y2: Int) {
        // 需要 accessibility service
    }

    private fun injectKey(keyCode: Int) {
        val event = KeyEvent(KeyEvent.ACTION_DOWN, keyCode)
        // 需要 accessibility service 或 root 注入
    }

    private suspend fun captureScreen(): Bitmap? {
        // 使用 MediaProjectionManager 截图
        // 需要先请求权限
        return null
    }

    // 上报命令结果到 Hub
    fun reportResult(commandId: String, success: Boolean, result: JSONObject) {
        scope.launch(Dispatchers.IO) {
            try {
                val deviceId = getDeviceId()
                val url = URL("$hubUrl/api/v2/mdm/devices/$deviceId/command-result")
                val connection = url.openConnection() as HttpURLConnection
                connection.requestMethod = "POST"
                connection.setRequestProperty("Content-Type", "application/json")
                connection.doOutput = true

                val response = JSONObject().apply {
                    put("commandId", commandId)
                    put("success", success)
                    put("result", result)
                    put("timestamp", System.currentTimeMillis())
                }

                connection.outputStream.use { os ->
                    os.write(response.toString().toByteArray())
                }

                connection.responseCode
            } catch (e: Exception) {
                Log.e("MDMCommandHandler", "Failed to report result", e)
            }
        }
    }

    private fun getDeviceId(): String {
        val prefs = context.getSharedPreferences("freekiosk", Context.MODE_PRIVATE)
        return prefs.getString("device_id", "unknown") ?: "unknown"
    }

    private val hubUrl: String
        get() {
            val prefs = context.getSharedPreferences("freekiosk", Context.MODE_PRIVATE)
            return prefs.getString("hub_url", "http://localhost:8081") ?: "http://localhost:8081"
        }
}
```

---

#### 第 5-6 周：应用安装服务

##### 2.2 应用安装模块

**文件**: `android/app/src/main/java/com/freekiosk/services/AppInstallService.kt` (NEW)

```kotlin
package com.freekiosk.services

import android.app.admin.DevicePolicyManager
import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.util.Log
import kotlinx.coroutines.*
import java.io.File
import java.io.FileOutputStream
import java.net.HttpURLConnection
import java.net.URL

class AppInstallService(private val context: Context) {

    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private val devicePolicyManager = context.getSystemService(Context.DEVICE_POLICY_SERVICE) as DevicePolicyManager
    private val adminComponent = ComponentName(context, FreeKioskDeviceAdminReceiver::class.java)

    companion object {
        private const val TAG = "AppInstallService"
    }

    // 安装应用
    fun installApp(packageUrl: String, callback: (Result) -> Unit) {
        scope.launch {
            try {
                // 1. 下载 APK
                val apkFile = downloadAPK(packageUrl)
                if (apkFile == null) {
                    withContext(Dispatchers.Main) {
                        callback(Result.failure("Download failed"))
                    }
                    return@launch
                }

                // 2. 验证 APK
                val packageInfo = context.packageManager.getPackageArchiveInfo(
                    apkFile.absolutePath,
                    PackageManager.GET_PERMISSIONS
                )
                if (packageInfo == null) {
                    withContext(Dispatchers.Main) {
                        callback(Result.failure("Invalid APK"))
                    }
                    return@launch
                }

                // 3. 安装 APK
                val success = installAPK(apkFile)
                if (success) {
                    // 4. 启用应用（如果是 Kiosk 应用）
                    enableKioskApp(packageInfo.packageName)

                    withContext(Dispatchers.Main) {
                        callback(Result.success(mapOf(
                            "package" to packageInfo.packageName,
                            "version" to packageInfo.versionName
                        )))
                    }
                } else {
                    withContext(Dispatchers.Main) {
                        callback(Result.failure("Installation failed"))
                    }
                }

                // 5. 清理临时文件
                apkFile.delete()
            } catch (e: Exception) {
                Log.e(TAG, "Install failed", e)
                withContext(Dispatchers.Main) {
                    callback(Result.failure(e.message ?: "Installation failed"))
                }
            }
        }
    }

    // 卸载应用
    fun uninstallApp(packageName: String, callback: (Result) -> Unit) {
        scope.launch {
            try {
                val success = uninstallAPK(packageName)
                withContext(Dispatchers.Main) {
                    if (success) {
                        callback(Result.success(mapOf("package" to packageName)))
                    } else {
                        callback(Result.failure("Uninstallation failed"))
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    callback(Result.failure(e.message ?: "Uninstallation failed"))
                }
            }
        }
    }

    private suspend fun downloadAPK(url: String): File? {
        return withContext(Dispatchers.IO) {
            try {
                val connection = URL(url).openConnection() as HttpURLConnection
                connection.connect()

                if (connection.responseCode !in 200..299) {
                    return@withContext null
                }

                val inputStream = connection.inputStream
                val tempFile = File(context.cacheDir, "temp_${System.currentTimeMillis()}.apk")

                FileOutputStream(tempFile).use { fos ->
                    inputStream.copyTo(fos)
                }

                tempFile
            } catch (e: Exception) {
                Log.e(TAG, "Download failed", e)
                null
            }
        }
    }

    private fun installAPK(file: File): Boolean {
        return try {
            val intent = Intent(Intent.ACTION_VIEW).apply {
                flags = Intent.FLAG_ACTIVITY_NEW_TASK
                setDataAndType(
                    Uri.fromFile(file),
                    "application/vnd.android.package-archive"
                )
            }

            if (intent.resolveActivity(context.packageManager) != null) {
                context.startActivity(intent)
                true
            } else {
                false
            }
        } catch (e: Exception) {
            Log.e(TAG, "Install failed", e)
            false
        }
    }

    private fun uninstallAPK(packageName: String): Boolean {
        return try {
            // 需要 Device Owner 权限
            if (devicePolicyManager.isAdminActive(adminComponent)) {
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
                    devicePolicyManager.removeActiveAdmin(adminComponent)
                }
                // 实际调用 packageManager.deletePackage
                true
            } else {
                false
            }
        } catch (e: Exception) {
            Log.e(TAG, "Uninstall failed", e)
            false
        }
    }

    private fun enableKioskApp(packageName: String) {
        // 将应用设为 Kiosk 模式可用
        try {
            val prefs = context.getSharedPreferences("freekiosk_kiosk", Context.MODE_PRIVATE)
            val kioskApps = prefs.getStringSet("kiosk_apps", mutableSetOf()) ?: mutableSetOf()
            kioskApps.add(packageName)
            prefs.edit().putStringSet("kiosk_apps", kioskApps).apply()
        } catch (e: Exception) {
            Log.e(TAG, "Failed to enable kiosk app", e)
        }
    }
}
```

---

#### 第 7-8 周：配置应用服务

##### 2.3 配置同步模块

**文件**: `android/app/src/main/java/com/freekiosk/services/ConfigSyncService.kt` (NEW)

```kotlin
package com.freekiosk.services

import android.content.Context
import android.content.SharedPreferences
import android.util.Log
import kotlinx.coroutines.*
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL

class ConfigSyncService(private val context: Context) {

    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private val prefs: SharedPreferences = context.getSharedPreferences("freekiosk_config", Context.MODE_PRIVATE)

    companion object {
        private const val TAG = "ConfigSyncService"
        private const val KEY_CURRENT_CONFIG_ID = "current_config_id"
        private const val KEY_CONFIG_HASH = "config_hash"
    }

    // 拉取设备配置
    fun pullConfiguration(callback: (Result) -> Unit) {
        scope.launch {
            try {
                val deviceId = getDeviceId()
                val url = URL("$hubUrl/api/v2/mdm/devices/$deviceId/configuration")

                val connection = url.openConnection() as HttpURLConnection
                connection.requestMethod = "GET"
                connection.setRequestProperty("Accept", "application/json")

                if (connection.responseCode == 200) {
                    val response = connection.inputStream.bufferedReader().readText()
                    val config = JSONObject(response)

                    // 应用配置
                    applyConfiguration(config)

                    // 更新配置哈希
                    val newHash = config.optString("hash", "")
                    prefs.edit()
                        .putString(KEY_CURRENT_CONFIG_ID, config.optString("id"))
                        .putString(KEY_CONFIG_HASH, newHash)
                        .apply()

                    withContext(Dispatchers.Main) {
                        callback(Result.success(config))
                    }
                } else {
                    withContext(Dispatchers.Main) {
                        callback(Result.failure("Failed to get configuration: ${connection.responseCode}"))
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "Config sync failed", e)
                withContext(Dispatchers.Main) {
                    callback(Result.failure(e.message ?: "Config sync failed"))
                }
            }
        }
    }

    // 应用配置到设备
    private fun applyConfiguration(config: JSONObject) {
        // 解析配置项
        val settings = config.optJSONObject("settings") ?: return

        // 应用锁屏设置
        if (settings.has("blockStatusBar")) {
            val blockStatusBar = settings.getBoolean("blockStatusBar")
            prefs.edit().putBoolean("block_status_bar", blockStatusBar).apply()
        }

        // 系统更新策略
        if (settings.has("systemUpdateType")) {
            val updateType = settings.getInt("systemUpdateType")
            prefs.edit().putInt("system_update_type", updateType).apply()
        }

        // 密码策略
        if (settings.has("passwordPolicy")) {
            val passwordPolicy = settings.getJSONObject("passwordPolicy")
            prefs.edit().putString("password_policy", passwordPolicy.toString()).apply()
        }

        // 应用白名单
        if (settings.has("appWhitelist")) {
            val whitelist = settings.getJSONArray("appWhitelist")
            val apps = mutableSetOf<String>()
            for (i in 0 until whitelist.length()) {
                apps.add(whitelist.getString(i))
            }
            prefs.edit().putStringSet("app_whitelist", apps).apply()
        }

        // 网络过滤规则
        if (settings.has("networkRules")) {
            val rules = settings.getJSONArray("networkRules")
            prefs.edit().putString("network_rules", rules.toString()).apply()
        }

        // 广播通知应用配置变更
        context.sendBroadcast(Intent("com.freekiosk.CONFIG_UPDATED").apply {
            putExtra("config_id", config.optString("id"))
        })
    }

    // 检查配置是否更新
    fun checkConfigUpdate(callback: (Boolean) -> Unit) {
        scope.launch {
            try {
                val deviceId = getDeviceId()
                val currentHash = prefs.getString(KEY_CONFIG_HASH, "")

                val url = URL("$hubUrl/api/v2/mdm/devices/$deviceId/configuration/hash")
                val connection = url.openConnection() as HttpURLConnection
                connection.requestMethod = "GET"

                if (connection.responseCode == 200) {
                    val response = connection.inputStream.bufferedReader().readText()
                    val result = JSONObject(response)
                    val serverHash = result.getString("hash")

                    withContext(Dispatchers.Main) {
                        callback(serverHash != currentHash)
                    }
                } else {
                    withContext(Dispatchers.Main) {
                        callback(false)
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "Check update failed", e)
                withContext(Dispatchers.Main) {
                    callback(false)
                }
            }
        }
    }

    private fun getDeviceId(): String {
        return context.getSharedPreferences("freekiosk", Context.MODE_PRIVATE)
            .getString("device_id", "unknown") ?: "unknown"
    }

    private val hubUrl: String
        get() = context.getSharedPreferences("freekiosk", Context.MODE_PRIVATE)
            .getString("hub_url", "http://localhost:8081") ?: "http://localhost:8081"
}
```

---

## 第二阶段：企业功能（第 9-16 周）

### Hub 端实现方案

#### 第 9-10 周：地理围栏

##### 3.1.1 围栏 Model

**文件**: `internal/models/geofence.go` (NEW)

```go
package models

import "time"

// Geofence 地理围栏
type Geofence struct {
    ID                 string    `json:"id" db:"id"`
    Name               string    `json:"name" db:"name"`
    Description        string    `json:"description" db:"description"`
    Latitude           float64   `json:"latitude" db:"latitude"`
    Longitude          float64   `json:"longitude" db:"longitude"`
    Radius             int       `json:"radius" db:"radius"` // 米
    EnterNotification  bool      `json:"enter_notification" db:"enter_notification"`
    ExitNotification   bool      `json:"exit_notification" db:"exit_notification"`
    Active            bool      `json:"active" db:"active"`
    TenantID           string    `json:"tenant_id" db:"tenant_id"`
    CreatedAt          int64     `json:"created_at" db:"created_at"`
    UpdatedAt          int64     `json:"updated_at" db:"updated_at"`
}

// GeofenceEvent 围栏事件
type GeofenceEvent struct {
    ID          string `json:"id" db:"id"`
    GeofenceID  string `json:"geofence_id" db:"geofence_id"`
    DeviceID    string `json:"device_id" db:"device_id"`
    EventType   string `json:"event_type" db:"event_type"` // enter, exit
    Latitude    float64 `json:"latitude" db:"latitude"`
    Longitude   float64 `json:"longitude" db:"longitude"`
    TriggeredAt int64   `json:"triggered_at" db:"triggered_at"`
}

// IsPointInGeofence 判断点是否在围栏内
func IsPointInGeofence(lat, lng float64, center *Geofence) bool {
    // 使用 Haversine 公式计算距离
    const earthRadius = 6371000 // 米

    lat1 := toRadians(center.Latitude)
    lat2 := toRadians(lat)
    dLat := toRadians(lat - center.Latitude)
    dLng := toRadians(lng - center.Longitude)

    a := math.Sin(dLat/2)*math.Sin(dLat/2) +
        math.Cos(lat1)*math.Cos(lat2)*
            math.Sin(dLng/2)*math.Sin(dLng/2)
    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

    distance := earthRadius * c
    return distance <= float64(center.Radius)
}

func toRadians(deg float64) float64 {
    return deg * math.Pi / 180
}
```

##### 3.1.2 围栏服务

**文件**: `internal/services/geofence_service.go` (NEW)

```go
package services

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/pseudolution/freekiosk-hub/internal/models"
    "github.com/pseudolution/freekiosk-hub/internal/repositories"
)

type GeofenceService interface {
    // 围栏管理
    CreateGeofence(ctx context.Context, tenantID string, req *CreateGeofenceRequest) (*models.Geofence, error)
    UpdateGeofence(ctx context.Context, geofenceID string, req *UpdateGeofenceRequest) error
    DeleteGeofence(ctx context.Context, geofenceID string) error
    ListGeofences(ctx context.Context, tenantID string) ([]*models.Geofence, error)
    GetGeofence(ctx context.Context, geofenceID string) (*models.Geofence, error)

    // 设备分配
    AssignDevices(ctx context.Context, geofenceID string, deviceIDs []string) error
    GetGeofenceDevices(ctx context.Context, geofenceID string) ([]*models.MDMDevice, error)

    // 事件处理
    ProcessLocationUpdate(ctx context.Context, deviceID string, lat, lng float64) error
    GetGeofenceEvents(ctx context.Context, geofenceID string, limit int) ([]*models.GeofenceEvent, error)
}

type geofenceService struct {
    geofenceRepo repositories.GeofenceRepository
    deviceRepo   repositories.MDMTabletRepository
    mqttSvc      *MQTTService
    notifySvc    *NotificationService
}

type CreateGeofenceRequest struct {
    Name              string  `json:"name"`
    Description       string  `json:"description,omitempty"`
    Latitude          float64 `json:"latitude"`
    Longitude         float64 `json:"longitude"`
    Radius            int     `json:"radius"`
    EnterNotification bool    `json:"enter_notification"`
    ExitNotification  bool    `json:"exit_notification"`
}

func (s *geofenceService) ProcessLocationUpdate(ctx context.Context, deviceID string, lat, lng float64) error {
    // 获取设备所属租户
    device, err := s.deviceRepo.GetDeviceByID(deviceID)
    if err != nil {
        return err
    }

    // 获取租户所有活跃围栏
    geofences, err := s.geofenceRepo.ListActiveGeofences(device.TenantID)
    if err != nil {
        return err
    }

    // 检查每个围栏
    for _, fence := range geofences {
        isInside := models.IsPointInGeofence(lat, lng, fence)

        // 获取设备之前是否在围栏内
        wasInside, err := s.geofenceRepo WasDeviceInside(fence.ID, deviceID)
        if err != nil {
            continue
        }

        now := time.Now().Unix()

        // 进入围栏
        if isInside && !wasInside {
            // 记录事件
            event := &models.GeofenceEvent{
                ID:         generateUUID(),
                GeofenceID: fence.ID,
                DeviceID:   deviceID,
                EventType:  "enter",
                Latitude:   lat,
                Longitude:  lng,
                TriggeredAt: now,
            }
            s.geofenceRepo.RecordEvent(event)

            // 发送通知
            if fence.EnterNotification {
                s.notifySvc.NotifyGeofenceEvent(device.TenantID, deviceID, fence, "enter")
            }

            // 更新设备状态
            s.geofenceRepo.SetDeviceInside(fence.ID, deviceID, true)
        }

        // 离开围栏
        if !isInside && wasInside {
            event := &models.GeofenceEvent{
                ID:         generateUUID(),
                GeofenceID: fence.ID,
                DeviceID:   deviceID,
                EventType:  "exit",
                Latitude:   lat,
                Longitude:  lng,
                TriggeredAt: now,
            }
            s.geofenceRepo.RecordEvent(event)

            if fence.ExitNotification {
                s.notifySvc.NotifyGeofenceEvent(device.TenantID, deviceID, fence, "exit")
            }

            s.geofenceRepo.SetDeviceInside(fence.ID, deviceID, false)
        }
    }

    // 更新设备最后位置
    return s.deviceRepo.UpdateLocation(deviceID, lat, lng, time.Now().Unix())
}
```

---

#### 第 11-12 周：远程控制

##### 3.2.1 远程会话 Model

**文件**: `internal/models/remote_session.go` (NEW)

```
package models

import "time"

// RemoteSession 远程控制会话
type RemoteSession struct {
    ID          string    `json:"id" db:"id"`
    DeviceID    string    `json:"device_id" db:"device_id"`
    UserID      string    `json:"user_id" db:"user_id"`
    Status      string    `json:"status" db:"status"` // pending, connecting, connected, disconnected, error, closed
    WebRTCData  string    `json:"webrtc_data" db:"webrtc_data"` // JSON
    ErrorMsg    string    `json:"error_msg" db:"error_msg"`
    StartedAt   *int64     `json:"started_at" db:"started_at"`
    EndedAt     *int64     `json:"ended_at" db:"ended_at"`
    CreatedAt   int64     `json:"created_at" db:"created_at"`
}

// RemoteSessionStatus 会话状态
const (
    SessionStatusPending      = "pending"
    SessionStatusConnecting   = "connecting"
    SessionStatusConnected    = "connected"
    SessionStatusDisconnected = "disconnected"
    SessionStatusError        = "error"
    SessionStatusClosed       = "closed"
)
```

##### 3.2.2 WebRTC 信令

**文件**: `internal/services/remote_control_service.go` (NEW)

```go
package services

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/pseudolution/freekiosk-hub/internal/models"
    "github.com/pseudolution/freekiosk-hub/internal/repositories"
)

type RemoteControlService interface {
    // 会话管理
    CreateSession(ctx context.Context, userID, deviceID string) (*models.RemoteSession, error)
    GetSession(ctx context.Context, sessionID string) (*models.RemoteSession, error)
    CloseSession(ctx context.Context, sessionID string) error

    // WebRTC 信令
    UpdateWebRTCData(ctx context.Context, sessionID string, data *WebRTCData) error
    GetWebRTCOffer(ctx context.Context, sessionID string) (*WebRTCSignal, error)
    HandleWebRTCAnswer(ctx context.Context, sessionID string, answer *WebRTCSignal) error
}

type WebRTCData struct {
    Type      string `json:"type"` // offer, answer, candidate
    SDP       string `json:"sdp,omitempty"`
    Candidate string `json:"candidate,omitempty"`
    Mid       string `json:"mid,omitempty"`
}

type WebRTCSignal struct {
    SessionID string      `json:"session_id"`
    From      string      `json:"from"` // "hub" or "device"
    To        string      `json:"to"`
    Signal    *WebRTCData `json:"signal"`
    Timestamp int64       `json:"timestamp"`
}

func (s *remoteControlService) CreateSession(ctx context.Context, userID, deviceID string) (*models.RemoteSession, error) {
    session := &models.RemoteSession{
        ID:       generateUUID(),
        DeviceID: deviceID,
        UserID:   userID,
        Status:   models.SessionStatusPending,
        CreatedAt: time.Now().Unix(),
    }

    if err := s.sessionRepo.Create(session); err != nil {
        return nil, err
    }

    // 通过 MQTT 发送会话请求到设备
    s.mqttSvc.PublishToDevice(deviceID, "remote_control/request", map[string]interface{}{
        "session_id": session.ID,
        "user_id":    userID,
        "timestamp":  session.CreatedAt,
    })

    return session, nil
}

func (s *remoteControlService) UpdateWebRTCData(ctx context.Context, sessionID string, data *WebRTCData) error {
    session, err := s.sessionRepo.Get(sessionID)
    if err != nil {
        return err
    }

    signal := &WebRTCSignal{
        SessionID: sessionID,
        From:      "hub",
        To:        session.DeviceID,
        Signal:    data,
        Timestamp: time.Now().Unix(),
    }

    // 通过 MQTT 发送 WebRTC 信令
    signalJSON, _ := json.Marshal(signal)
    s.mqttSvc.PublishToDevice(session.DeviceID, "remote_control/webrtc", signalJSON)

    // 更新会话状态
    if data.Type == "answer" {
        session.Status = models.SessionStatusConnected
        now := time.Now().Unix()
        session.StartedAt = &now
        return s.sessionRepo.Update(session)
    }

    return nil
}
```

---

#### 第 13-14 周：消息推送

##### 3.3.1 消息 Model

**文件**: `internal/models/push_message.go` (NEW)

```go
package models

import "time"

// PushMessage 推送消息
type PushMessage struct {
    ID          string    `json:"id" db:"id"`
    DeviceID   string    `json:"device_id" db:"device_id"`
    MessageType string    `json:"message_type" db:"message_type"` // text, notification, command
    Title       string    `json:"title" db:"title"`
    Body        string    `json:"body" db:"body"`
    Payload     string    `json:"payload" db:"payload"` // JSON
    Status      string    `json:"status" db:"status"`   // pending, sent, delivered, read, failed
    Priority    int       `json:"priority" db:"priority"` // 0=low, 1=normal, 2=high
    ExpiresAt   *int64    `json:"expires_at" db:"expires_at"`
    CreatedAt   int64     `json:"created_at" db:"created_at"`
    SentAt      *int64    `json:"sent_at" db:"sent_at"`
    DeliveredAt *int64    `json:"delivered_at" db:"delivered_at"`
    ReadAt      *int64    `json:"read_at" db:"read_at"`
}

// PushStatus 消息状态
const (
    PushStatusPending   = "pending"
    PushStatusSent      = "sent"
    PushStatusDelivered = "delivered"
    PushStatusRead      = "read"
    PushStatusFailed    = "failed"
)

// ScheduledPush 定时推送任务
type ScheduledPush struct {
    ID          string    `json:"id" db:"id"`
    Name        string    `json:"name" db:"name"`
    Message     *PushMessage `json:"message" db:"message"`
    ScheduleTime int64     `json:"schedule_time" db:"schedule_time"`
    Repeat      string    `json:"repeat" db:"repeat"` // none, daily, weekly
    Active      bool      `json:"active" db:"active"`
    TenantID    string    `json:"tenant_id" db:"tenant_id"`
    CreatedAt   int64     `json:"created_at" db:"created_at"`
}
```

##### 3.3.2 推送服务

**文件**: `internal/services/push_service.go` (NEW)

```go
package services

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/pseudolution/freekiosk-hub/internal/models"
    "github.com/pseudolution/freekiosk-hub/internal/repositories"
)

type PushService interface {
    // 消息发送
    SendMessage(ctx context.Context, req *SendMessageRequest) (*models.PushMessage, error)
    SendBatchMessage(ctx context.Context, req *SendBatchMessageRequest) ([]*models.PushMessage, error)

    // 消息管理
    GetMessage(ctx context.Context, messageID string) (*models.PushMessage, error)
    ListMessages(ctx context.Context, deviceID string, limit int) ([]*models.PushMessage, error)
    DeleteMessage(ctx context.Context, messageID string) error

    // 消息状态更新（设备回调）
    MarkDelivered(ctx context.Context, messageID string) error
    MarkRead(ctx context.Context, messageID string) error

    // 定时任务
    CreateScheduledPush(ctx context.Context, req *CreateScheduledPushRequest) (*models.ScheduledPush, error)
    ListScheduledPushes(ctx context.Context, tenantID string) ([]*models.ScheduledPush, error)
    DeleteScheduledPush(ctx context.Context, scheduleID string) error
}

type SendMessageRequest struct {
    DeviceID    string `json:"device_id"`
    MessageType string `json:"message_type"` // text, notification, command
    Title       string `json:"title"`
    Body        string `json:"body"`
    Payload     string `json:"payload,omitempty"` // JSON string
    Priority    int    `json:"priority,omitempty"`
}

func (s *pushService) SendMessage(ctx context.Context, req *SendMessageRequest) (*models.PushMessage, error) {
    message := &models.PushMessage{
        ID:          generateUUID(),
        DeviceID:    req.DeviceID,
        MessageType: req.MessageType,
        Title:       req.Title,
        Body:        req.Body,
        Payload:     req.Payload,
        Status:      models.PushStatusPending,
        Priority:    req.Priority,
        CreatedAt:   time.Now().Unix(),
    }

    if err := s.messageRepo.Create(message); err != nil {
        return nil, err
    }

    // 通过 MQTT 发送
    mqttPayload := map[string]interface{}{
        "id":           message.ID,
        "type":         message.MessageType,
        "title":        message.Title,
        "body":         message.Body,
        "payload":      message.Payload,
        "created_at":   message.CreatedAt,
    }

    if err := s.mqttSvc.PublishToDevice(message.DeviceID, "push/message", mqttPayload); err != nil {
        message.Status = models.PushStatusFailed
        s.messageRepo.Update(message)
        return nil, fmt.Errorf("failed to send push: %w", err)
    }

    message.Status = models.PushStatusSent
    now := time.Now().Unix()
    message.SentAt = &now
    s.messageRepo.Update(message)

    return message, nil
}
```

---

#### 第 15-16 周：用户和 RBAC

##### 3.4.1 用户 Model 扩展

**文件**: `internal/models/user.go` (扩展现有)

```go
// UserRole 用户角色
type UserRole struct {
    ID          string   `json:"id" db:"id"`
    Name        string   `json:"name" db:"name"`
    Description string   `json:"description" db:"description"`
    Permissions []string `json:"permissions" db:"permissions"` // JSON array
    IsSystem    bool     `json:"is_system" db:"is_system"`   // 系统内置角色不可删除
    TenantID    string   `json:"tenant_id" db:"tenant_id"`
    CreatedAt   int64    `json:"created_at" db:"created_at"`
    UpdatedAt   int64    `json:"updated_at" db:"updated_at"`
}

// Permission constants
const (
    PermissionDeviceRead   = "device:read"
    PermissionDeviceWrite  = "device:write"
    PermissionDeviceDelete = "device:delete"
    PermissionConfigRead   = "config:read"
    PermissionConfigWrite  = "config:write"
    PermissionAppRead     = "app:read"
    PermissionAppWrite    = "app:write"
    PermissionAppInstall  = "app:install"
    PermissionUserRead    = "user:read"
    PermissionUserWrite   = "user:write"
    PermissionAdmin       = "admin:*"
)

// DefaultRoles 默认角色
var DefaultRoles = []*UserRole{
    {
        Name:        "Administrator",
        Description: "Full system access",
        Permissions: []string{PermissionAdmin},
        IsSystem:    true,
    },
    {
        Name:        "Operator",
        Description: "Device and configuration management",
        Permissions: []string{
            PermissionDeviceRead, PermissionDeviceWrite,
            PermissionConfigRead, PermissionConfigWrite,
            PermissionAppRead, PermissionAppWrite,
        },
        IsSystem: true,
    },
    {
        Name:        "Viewer",
        Description: "Read-only access",
        Permissions: []string{
            PermissionDeviceRead, PermissionConfigRead, PermissionAppRead,
        },
        IsSystem: true,
    },
}
```

##### 3.4.2 权限中间件

**文件**: `internal/api/rbac_middleware.go` (NEW)

```go
package api

import (
    "net/http"
    "strings"

    "github.com/labstack/echo/v4"
    "github.com/pseudolution/freekiosk-hub/internal/models"
)

// RBACMiddleware 基于角色的访问控制中间件
func RBACMiddleware(requiredPermission string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            user := c.Get("user").(*models.User)
            role := c.Get("role").(*models.UserRole)

            // 检查权限
            if !hasPermission(role, requiredPermission) {
                return echo.NewHTTPError(http.StatusForbidden, "Permission denied")
            }

            return next(c)
        }
    }
}

func hasPermission(role *models.UserRole, permission string) bool {
    for _, p := range role.Permissions {
        if p == permission {
            return true
        }
        // admin:* 匹配所有权限
        if p == models.PermissionAdmin {
            return true
        }
        // wildcard 匹配
        if strings.HasSuffix(p, ":*") {
            prefix := strings.TrimSuffix(p, "*")
            if strings.HasPrefix(permission, prefix) {
                return true
            }
        }
    }
    return false
}

// PermissionCheck 权限检查辅助函数
func PermissionCheck(c echo.Context, permission string) bool {
    role := c.Get("role").(*models.UserRole)
    return hasPermission(role, permission)
}
```

---

### Android 端实现方案

#### 第 9-10 周：地理围栏检测

##### 4.1 围栏检测服务

**文件**: `android/app/src/main/java/com/freekiosk/services/GeofenceService.kt` (NEW)

```kotlin
package com.freekiosk.services

import android.Manifest
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.location.Location
import android.os.Build
import android.util.Log
import androidx.core.app.ActivityCompat
import com.google.android.gms.location.*
import kotlinx.coroutines.*
import kotlin.math.*

class GeofenceService(private val context: Context) {

    private val scope = CoroutineScope(Dispatchers.Main + SupervisorJob())
    private val fusedLocationClient = LocationServices.getFusedLocationProviderClient(context)
    private val geofencingClient = LocationServices.getGeofencingClient(context)

    private var currentLocation: Location? = null
    private var monitoredGeofences: List<GeofenceData> = emptyList()
    private var deviceInsideFences: MutableMap<String, Boolean> = mutableMapOf()

    data class GeofenceData(
        val id: String,
        val name: String,
        val latitude: Double,
        val longitude: Double,
        val radius: Float
    )

    companion object {
        private const val TAG = "GeofenceService"
        private const val GEOFENCE_TRANSITIONTypes =
            Geofence.GEOFENCE_TRANSITION_ENTER or Geofence.GEOFENCE_TRANSITION_EXIT
    }

    // 拉取围栏配置
    fun fetchGeofences(callback: (Result) -> Unit) {
        scope.launch(Dispatchers.IO) {
            try {
                val deviceId = getDeviceId()
                val url = URL("$hubUrl/api/v2/mdm/devices/$deviceId/geofences")

                val connection = url.openConnection() as HttpURLConnection
                connection.requestMethod = "GET"

                if (connection.responseCode == 200) {
                    val response = connection.inputStream.bufferedReader().readText()
                    val jsonArray = org.json.JSONArray(response)

                    val fences = mutableListOf<GeofenceData>()
                    for (i in 0 until jsonArray.length()) {
                        val obj = jsonArray.getJSONObject(i)
                        fences.add(GeofenceData(
                            id = obj.getString("id"),
                            name = obj.getString("name"),
                            latitude = obj.getDouble("latitude"),
                            longitude = obj.getDouble("longitude"),
                            radius = obj.getInt("radius").toFloat()
                        ))
                    }

                    monitoredGeofences = fences

                    // 注册 Google 围栏（如果可用）
                    registerGoogleGeofences(fences)

                    withContext(Dispatchers.Main) {
                        callback(Result.success(fences))
                    }
                } else {
                    withContext(Dispatchers.Main) {
                        callback(Result.failure("Failed to fetch geofences"))
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "Fetch geofences failed", e)
                withContext(Dispatchers.Main) {
                    callback(Result.failure(e.message ?: "Failed"))
                }
            }
        }
    }

    // 启动位置更新监听
    fun startLocationUpdates() {
        if (ActivityCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION)
            != PackageManager.PERMISSION_GRANTED) {
            return
        }

        val locationRequest = LocationRequest.Builder(
            Priority.PRIORITY_HIGH_ACCURACY,
            30000 // 30秒
        ).apply {
            setMinUpdateIntervalMillis(15000)
        }.build()

        fusedLocationClient.requestLocationUpdates(
            locationRequest,
            locationCallback,
            context.mainLooper
        )
    }

    private val locationCallback = object : LocationCallback() {
        override fun onLocationResult(result: LocationResult) {
            result.lastLocation?.let { location ->
                currentLocation = location
                checkGeofences(location)
            }
        }
    }

    // 检查围栏状态
    private fun checkGeofences(location: Location) {
        for (fence in monitoredGeofences) {
            val distance = calculateDistance(
                location.latitude, location.longitude,
                fence.latitude, fence.longitude
            )

            val isInside = distance <= fence.radius
            val wasInside = deviceInsideFences[fence.id] ?: false

            if (isInside != wasInside) {
                deviceInsideFences[fence.id] = isInside

                // 上报围栏事件
                reportGeofenceEvent(fence.id, if (isInside) "enter" else "exit",
                    location.latitude, location.longitude)
            }
        }
    }

    // 计算两点间距离（米）- Haversine 公式
    private fun calculateDistance(lat1: Double, lon1: Double, lat2: Double, lon2: Double): Float {
        val r = 6371000.0 // 地球半径（米）

        val dLat = Math.toRadians(lat2 - lat1)
        val dLon = Math.toRadians(lon2 - lon1)

        val a = sin(dLat / 2) * sin(dLat / 2) +
                cos(Math.toRadians(lat1)) * cos(Math.toRadians(lat2)) *
                sin(dLon / 2) * sin(dLon / 2)

        val c = 2 * atan2(sqrt(a), sqrt(1 - a))

        return (r * c).toFloat()
    }

    // 注册 Google 围栏（后台检测）
    private fun registerGoogleGeofences(geofences: List<GeofenceData>) {
        if (ActivityCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION)
            != PackageManager.PERMISSION_GRANTED) {
            return
        }

        val geofenceList = geofences.map { fence ->
            Geofence.Builder()
                .setRequestId(fence.id)
                .setCircularRegion(fence.latitude, fence.longitude, fence.radius)
                .setExpirationDuration(Geofence.NEVER_EXPIRE)
                .setTransitionTypes(GEOFENCE_TRANSITIONTypes)
                .build()
        }

        val pendingIntent = getGeofencePendingIntent()

        geofencingClient.addGeofences(geofenceList, pendingIntent)
            .addOnSuccessListener {
                Log.d(TAG, "Geofences registered: ${geofences.size}")
            }
            .addOnFailureListener { e ->
                Log.e(TAG, "Failed to register geofences", e)
            }
    }

    private fun getGeofencePendingIntent(): PendingIntent {
        val intent = Intent(context, GeofenceBroadcastReceiver::class.java)
        return PendingIntent.getBroadcast(
            context,
            0,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_MUTABLE
        )
    }

    // 上报围栏事件到 Hub
    private fun reportGeofenceEvent(geofenceId: String, eventType: String, lat: Double, lng: Double) {
        scope.launch(Dispatchers.IO) {
            try {
                val deviceId = getDeviceId()
                val eventData = mapOf(
                    "geofence_id" to geofenceId,
                    "device_id" to deviceId,
                    "event_type" to eventType,
                    "latitude" to lat,
                    "longitude" to lng,
                    "timestamp" to System.currentTimeMillis()
                )

                val url = URL("$hubUrl/api/v2/mdm/devices/$deviceId/geofence-event")
                val connection = url.openConnection() as HttpURLConnection
                connection.requestMethod = "POST"
                connection.setRequestProperty("Content-Type", "application/json")
                connection.doOutput = true

                connection.outputStream.use { os ->
                    os.write(org.json.JSONObject(eventData).toString().toByteArray())
                }

                Log.d(TAG, "Geofence event reported: $eventType")
            } catch (e: Exception) {
                Log.e(TAG, "Failed to report geofence event", e)
            }
        }
    }

    private fun getDeviceId(): String {
        return context.getSharedPreferences("freekiosk", Context.MODE_PRIVATE)
            .getString("device_id", "unknown") ?: "unknown"
    }

    private val hubUrl: String
        get() = context.getSharedPreferences("freekiosk", Context.MODE_PRIVATE)
            .getString("hub_url", "http://localhost:8081") ?: "http://localhost:8081"
}
```

---

#### 第 11-12 周：远程控制接收

##### 4.2 远程控制服务

**文件**: `android/app/src/main/java/com/freekiosk/services/RemoteControlService.kt` (NEW)

```kotlin
package com.freekiosk.services

import android.app.Activity
import android.content.Context
import android.util.Log
import android.view.WindowManager
import com.google.gson.Gson
import kotlinx.coroutines.*
import org.webrtc.*
import java.io.DataOutputStream
import java.net.HttpURLConnection
import java.net.URL

class RemoteControlService(private val context: Context) {

    private val scope = CoroutineScope(Dispatchers.Main + SupervisorJob())
    private var webSocket: WebSocket? = null
    private var peerConnection: PeerConnection? = null
    private var currentSessionId: String? = null
    private var isControlling = false

    private val eglBase = EglBase.create()

    data class SessionRequest(
        val session_id: String,
        val user_id: String,
        val timestamp: Long
    )

    data class WebRTCSignal(
        val session_id: String,
        val from: String,
        val to: String,
        val signal: SignalData
    )

    data class SignalData(
        val type: String,
        val sdp: String? = null,
        val candidate: String? = null,
        val sdpMid: String? = null,
        val sdpMLineIndex: Int? = null
    )

    companion object {
        private const val TAG = "RemoteControlService"
    }

    // 处理会话请求
    fun handleSessionRequest(request: SessionRequest, activity: Activity, callback: (Boolean) -> Unit) {
        // 显示确认对话框
        scope.launch {
            val approved = showApprovalDialog(activity, request.user_id)
            if (approved) {
                acceptSession(request.session_id)
                callback(true)
            } else {
                rejectSession(request.session_id)
                callback(false)
            }
        }
    }

    private fun showApprovalDialog(activity: Activity, userId: String): Boolean {
        // 实际需要显示 Dialog，这里简化
        return true
    }

    private fun acceptSession(sessionId: String) {
        currentSessionId = sessionId
        scope.launch(Dispatchers.IO) {
            // 发送 WebRTC offer
            createWebRTCOffer(sessionId)
        }
    }

    private fun rejectSession(sessionId: String) {
        scope.launch(Dispatchers.IO) {
            try {
                val deviceId = getDeviceId()
                val url = URL("$hubUrl/api/v2/mdm/remote-control/sessions/$sessionId/reject")
                val connection = url.openConnection() as HttpURLConnection
                connection.requestMethod = "POST"
                connection.setRequestProperty("Content-Type", "application/json")
                connection.doOutput = true

                val body = """{"device_id": "$deviceId"}"""
                connection.outputStream.use { os ->
                    os.write(body.toByteArray())
                }
            } catch (e: Exception) {
                Log.e(TAG, "Failed to reject session", e)
            }
        }
    }

    private fun createWebRTCOffer(sessionId: String) {
        val factory = PeerConnectionFactory.builder()
            .setVideoDecoderFactory(DefaultVideoDecoderFactory(eglBase.eglBaseContext))
            .setVideoEncoderFactory(DefaultVideoEncoderFactory(eglBase.eglBaseContext))
            .createPeerConnectionFactory()

        val config = PeerConnection.RTCConfiguration(arrayListOf(
            "stun:stun.l.google.com:19302"
        ))

        peerConnection = factory.createPeerConnection(config, object : PeerConnection.Observer {
            override fun onSignalingChange(state: PeerConnection.SignalingState?) {}
            override fun onIceConnectionChange(state: PeerConnection.IceConnectionState?) {
                if (state == PeerConnection.IceConnectionState.CONNECTED) {
                    isControlling = true
                    startScreenCapture()
                }
            }
            override fun onIceConnectionReceivingChange(receiving: Boolean) {}
            override fun onIceGatheringChange(state: PeerConnection.IceGatheringState?) {}
            override fun onIceCandidate(candidate: IceCandidate?) {
                candidate?.let { sendIceCandidate(sessionId, it) }
            }
            override fun onAddStream(stream: MediaStream?) {}
            override fun onRemoveStream(stream: MediaStream?) {}
            override fun onDataChannel(channel: DataChannel?) {}
            override fun onRenegotiationNeeded() {}
        })

        // 创建屏幕视频 track
        val videoCapturer = createScreenCapturer()
        val videoSource = factory.createVideoSource(videoCapturer)
        val videoTrack = factory.createVideoTrack("video", videoSource)

        // 添加 ICE candidate
        val constraints = MediaConstraints().apply {
            mandatory.add(MediaConstraints.KeyValuePair("OfferToReceiveAudio", "false"))
            mandatory.add(MediaConstraints.KeyValuePair("OfferToReceiveVideo", "true"))
        }

        peerConnection?.createOffer({ sdp ->
            peerConnection?.setLocalDescription(object : SdpObserver {
                override fun onSetFailure(error: String?) {}
                override fun onSetSuccess() {
                    sendWebRTCOffer(sessionId, sdp.description)
                }
                override fun onCreateSuccess(sdp: SessionDescription?) {}
                override fun onCreateFailure(error: String?) {}
            }, constraints)
        }, constraints)
    }

    private fun sendWebRTCOffer(sessionId: String, sdp: String) {
        scope.launch(Dispatchers.IO) {
            try {
                val signal = WebRTCSignal(
                    session_id = sessionId,
                    from = "device",
                    to = "hub",
                    signal = SignalData(type = "offer", sdp = sdp)
                )

                postToHub("/api/v2/mdm/remote-control/sessions/$sessionId/webrtc", signal)
            } catch (e: Exception) {
                Log.e(TAG, "Failed to send offer", e)
            }
        }
    }

    private fun sendIceCandidate(sessionId: String, candidate: IceCandidate) {
        scope.launch(Dispatchers.IO) {
            try {
                val signal = WebRTCSignal(
                    session_id = sessionId,
                    from = "device",
                    to = "hub",
                    signal = SignalData(
                        type = "candidate",
                        candidate = candidate.sdp,
                        sdpMid = candidate.sdpMid,
                        sdpMLineIndex = candidate.sdpMLineIndex
                    )
                )

                postToHub("/api/v2/mdm/remote-control/sessions/$sessionId/webrtc", signal)
            } catch (e: Exception) {
                Log.e(TAG, "Failed to send ICE candidate", e)
            }
        }
    }

    private fun startScreenCapture() {
        // 开始屏幕捕获并通过 WebRTC 发送
        isControlling = true
    }

    // 处理远程输入
    fun handleRemoteInput(input: SignalData) {
        when (input.type) {
            "click" -> {
                val x = input.sdp?.split(",")?.getOrNull(0)?.toIntOrNull() ?: 0
                val y = input.sdp?.split(",")?.getOrNull(1)?.toIntOrNull() ?: 0
                injectClick(x, y)
            }
            "swipe" -> {
                val coords = input.sdp?.split(",")?.mapNotNull { it.toIntOrNull() } ?: return
                if (coords.size >= 4) {
                    injectSwipe(coords[0], coords[1], coords[2], coords[3])
                }
            }
            "key" -> {
                input.sdpMLineIndex?.let { injectKey(it) }
            }
        }
    }

    private fun injectClick(x: Int, y: Int) {
        // 使用 accessibility service 或 root 注入点击
    }

    private fun injectSwipe(x1: Int, y1: Int, x2: Int, y2: Int) {
        // 使用 accessibility service 或 root 注入滑动
    }

    private fun injectKey(keyCode: Int) {
        // 使用 accessibility service 或 root 注入按键
    }

    private fun createScreenCapturer(): VideoCapturer {
        // 使用 ScreenCapturerAndroid
        return ScreenCapturerAndroid(
            android.media.projection.MediaProjectionManager::class.java.classLoader!! as android.media.projection.MediaProjection
        )
    }

    private suspend fun postToHub(endpoint: String, data: Any) {
        withContext(Dispatchers.IO) {
            val url = URL("$hubUrl$endpoint")
            val connection = url.openConnection() as HttpURLConnection
            connection.requestMethod = "POST"
            connection.setRequestProperty("Content-Type", "application/json")
            connection.doOutput = true

            connection.outputStream.use { os ->
                os.write(Gson().toJson(data).toByteArray())
            }
        }
    }

    private fun getDeviceId(): String {
        return context.getSharedPreferences("freekiosk", Context.MODE_PRIVATE)
            .getString("device_id", "unknown") ?: "unknown"
    }

    private val hubUrl: String
        get() = context.getSharedPreferences("freekiosk", Context.MODE_PRIVATE)
            .getString("hub_url", "http://localhost:8081") ?: "http://localhost:8081"
}
```

---

#### 第 13-14 周：消息接收

##### 4.3 消息接收服务

**文件**: `android/app/src/main/java/com/freekiosk/services/PushNotificationService.kt` (NEW)

```kotlin
package com.freekiosk.services

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Build
import android.util.Log
import androidx.core.app.NotificationCompat
import com.facebook.react.bridge.*
import kotlinx.coroutines.*
import org.json.JSONObject

class PushNotificationService(private val context: Context) {

    private val scope = CoroutineScope(Dispatchers.Main + SupervisorJob())
    private val notificationManager = context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager

    companion object {
        private const val TAG = "PushNotificationService"
        private const val CHANNEL_ID = "freekiosk_push"
        private const val CHANNEL_NAME = "FreeKiosk Notifications"
    }

    init {
        createNotificationChannel()
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                CHANNEL_NAME,
                NotificationManager.IMPORTANCE_HIGH
            ).apply {
                description = "Device push notifications"
                enableVibration(true)
            }
            notificationManager.createNotificationChannel(channel)
        }
    }

    // 处理推送消息
    fun handlePushMessage(message: JSONObject) {
        val messageId = message.optString("id")
        val type = message.optString("type")
        val title = message.optString("title")
        val body = message.optString("body")
        val payload = message.optString("payload", "{}")

        when (type) {
            "text" -> showTextNotification(messageId, title, body, payload)
            "notification" -> showSystemNotification(messageId, title, body)
            "command" -> handleCommand(payload)
        }

        // 发送已读回执
        reportDelivered(messageId)
    }

    private fun showTextNotification(messageId: String, title: String, body: String, payload: String) {
        val intent = Intent(context, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            putExtra("message_id", messageId)
            putExtra("payload", payload)
        }

        val pendingIntent = PendingIntent.getActivity(
            context,
            messageId.hashCode(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val notification = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(android.R.drawable.ic_dialog_info)
            .setContentTitle(title)
            .setContentText(body)
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .build()

        notificationManager.notify(messageId.hashCode(), notification)
    }

    private fun showSystemNotification(messageId: String, title: String, body: String) {
        // 显示系统通知（来自 Hub 的系统消息）
        showTextNotification(messageId, title, body, "{}")
    }

    private fun handleCommand(payload: String) {
        try {
            val command = JSONObject(payload)
            // 解析并执行命令
            // 与 MDMCommandHandler 集成
        } catch (e: Exception) {
            Log.e(TAG, "Failed to handle command", e)
        }
    }

    private fun reportDelivered(messageId: String) {
        scope.launch(Dispatchers.IO) {
            try {
                val deviceId = getDeviceId()
                val url = URL("$hubUrl/api/v2/mdm/messages/$messageId/delivered")
                val connection = url.openConnection() as HttpURLConnection
                connection.requestMethod = "POST"
                connection.setRequestProperty("Content-Type", "application/json")

                val body = """{"device_id": "$deviceId", "timestamp": ${System.currentTimeMillis()}}"""
                connection.outputStream.use { os ->
                    os.write(body.toByteArray())
                }
            } catch (e: Exception) {
                Log.e(TAG, "Failed to report delivered", e)
            }
        }
    }

    private fun getDeviceId(): String {
        return context.getSharedPreferences("freekiosk", Context.MODE_PRIVATE)
            .getString("device_id", "unknown") ?: "unknown"
    }

    private val hubUrl: String
        get() = context.getSharedPreferences("freekiosk", Context.MODE_PRIVATE)
            .getString("hub_url", "http://localhost:8081") ?: "http://localhost:8081"
}
```

---

#### 第 15-16 周：用户认证集成

##### 4.4 设备认证服务

**文件**: `android/app/src/main/java/com/freekiosk/services/DeviceAuthService.kt` (NEW)

```kotlin
package com.freekiosk.services

import android.content.Context
import android.content.SharedPreferences
import android.util.Log
import kotlinx.coroutines.*
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL

class DeviceAuthService(private val context: Context) {

    private val scope = CoroutineScope(Dispatchers.Main + SupervisorJob())
    private val prefs: SharedPreferences = context.getSharedPreferences("freekiosk_auth", Context.MODE_PRIVATE)

    companion object {
        private const val TAG = "DeviceAuthService"
        private const val KEY_DEVICE_TOKEN = "device_token"
        private const val KEY_TOKEN_EXPIRY = "token_expiry"
    }

    // 设备注册/认证
    fun authenticateDevice(callback: (Result) -> Unit) {
        scope.launch(Dispatchers.IO) {
            try {
                val deviceId = getDeviceId()
                val deviceInfo = collectDeviceInfo()

                val requestBody = JSONObject().apply {
                    put("device_id", deviceId)
                    put("device_info", deviceInfo)
                    put("hub_url", hubUrl)
                }

                val url = URL("$hubUrl/api/v2/mdm/auth/register")
                val connection = url.openConnection() as HttpURLConnection
                connection.requestMethod = "POST"
                connection.setRequestProperty("Content-Type", "application/json")
                connection.doOutput = true

                connection.outputStream.use { os ->
                    os.write(requestBody.toString().toByteArray())
                }

                if (connection.responseCode in 200..299) {
                    val response = connection.inputStream.bufferedReader().readText()
                    val json = JSONObject(response)

                    // 保存 token
                    val token = json.optString("token")
                    val expiresIn = json.optInt("expires_in", 86400 * 30) // 默认30天

                    prefs.edit()
                        .putString(KEY_DEVICE_TOKEN, token)
                        .putLong(KEY_TOKEN_EXPIRY, System.currentTimeMillis() / 1000 + expiresIn)
                        .apply()

                    Log.d(TAG, "Device authenticated successfully")

                    withContext(Dispatchers.Main) {
                        callback(Result.success(json.toMap()))
                    }
                } else {
                    withContext(Dispatchers.Main) {
                        callback(Result.failure("Authentication failed: ${connection.responseCode}"))
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "Authentication failed", e)
                withContext(Dispatchers.Main) {
                    callback(Result.failure(e.message ?: "Authentication failed"))
                }
            }
        }
    }

    // 刷新 token
    fun refreshToken(callback: (Result) -> Unit) {
        scope.launch(Dispatchers.IO) {
            try {
                val deviceId = getDeviceId()
                val currentToken = prefs.getString(KEY_DEVICE_TOKEN, null)

                if (currentToken == null) {
                    // 没有 token，需要重新认证
                    authenticateDevice(callback)
                    return@launch
                }

                val url = URL("$hubUrl/api/v2/mdm/auth/refresh")
                val connection = url.openConnection() as HttpURLConnection
                connection.requestMethod = "POST"
                connection.setRequestProperty("Content-Type", "application/json")
                connection.setRequestProperty("Authorization", "Bearer $currentToken")
                connection.doOutput = true

                val body = """{"device_id": "$deviceId"}"""
                connection.outputStream.use { os ->
                    os.write(body.toByteArray())
                }

                if (connection.responseCode in 200..299) {
                    val response = connection.inputStream.bufferedReader().readText()
                    val json = JSONObject(response)

                    val newToken = json.optString("token")
                    val expiresIn = json.optInt("expires_in", 86400 * 30)

                    prefs.edit()
                        .putString(KEY_DEVICE_TOKEN, newToken)
                        .putLong(KEY_TOKEN_EXPIRY, System.currentTimeMillis() / 1000 + expiresIn)
                        .apply()

                    withContext(Dispatchers.Main) {
                        callback(Result.success(json.toMap()))
                    }
                } else if (connection.responseCode == 401) {
                    // Token 过期，需要重新认证
                    authenticateDevice(callback)
                } else {
                    withContext(Dispatchers.Main) {
                        callback(Result.failure("Refresh failed: ${connection.responseCode}"))
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "Token refresh failed", e)
                withContext(Dispatchers.Main) {
                    callback(Result.failure(e.message ?: "Refresh failed"))
                }
            }
        }
    }

    // 获取有效 token
    fun getValidToken(callback: (String?) -> Unit) {
        val token = prefs.getString(KEY_DEVICE_TOKEN, null)
        val expiry = prefs.getLong(KEY_TOKEN_EXPIRY, 0)

        if (token == null || System.currentTimeMillis() / 1000 > expiry) {
            // Token 无效，刷新
            refreshToken { result ->
                when (result) {
                    is Result.Success -> callback(result.data["token"] as? String)
                    is Result.Failure -> callback(null)
                }
            }
        } else {
            callback(token)
        }
    }

    private fun collectDeviceInfo(): JSONObject {
        val pm = context.packageManager
        val packageInfo = pm.getPackageInfo(context.packageName, 0)

        return JSONObject().apply {
            put("model", android.os.Build.MODEL)
            put("manufacturer", android.os.Build.MANUFACTURER)
            put("os_version", android.os.Build.VERSION.RELEASE)
            put("sdk_version", android.os.Build.VERSION.SDK_INT)
            put("app_version", packageInfo.versionName)
            put("app_version_code", packageInfo.versionCode)
        }
    }

    private fun getDeviceId(): String {
        return context.getSharedPreferences("freekiosk", Context.MODE_PRIVATE)
            .getString("device_id", "unknown") ?: "unknown"
    }

    private val hubUrl: String
        get() = context.getSharedPreferences("freekiosk", Context.MODE_PRIVATE)
            .getString("hub_url", "http://localhost:8081") ?: "http://localhost:8081"
}
```

---

## 产品验收标准

### Phase 1 验收标准（第 1-8 周）

#### Hub 端验收

| 功能 | 验收标准 | 测试方法 |
|------|----------|----------|
| 设备注册 API | POST /api/v2/mdm/devices 返回 201，设备信息正确存储 | curl 测试 |
| 设备列表 API | GET /api/v2/mdm/devices?tenant_id=xxx 返回设备列表，分页正常 | curl 测试 |
| 设备搜索 | POST /api/v2/mdm/devices/search 支持 name/number/imei 模糊搜索 | curl 测试 |
| 设备详情 | GET /api/v2/mdm/devices/:id 返回完整设备信息 | curl 测试 |
| 设备更新 | PUT /api/v2/mdm/devices/:id 可更新 name/description/status | curl 测试 |
| 设备删除 | DELETE /api/v2/mdm/devices/:id 返回 204 | curl 测试 |
| 设备分组 | CRUD /api/v2/mdm/groups 正常 | curl 测试 |
| 设备分组分配 | POST /api/v2/mdm/devices/:id/group 可分配到分组 | curl 测试 |
| 设备锁定 | POST /api/v2/mdm/devices/:id/lock 发送 MQTT lock 命令 | 日志确认 |
| 设备重启 | POST /api/v2/mdm/devices/:id/reboot 发送 MQTT reboot 命令 | 日志确认 |
| 设备位置 | GET /api/v2/mdm/devices/:id/location 返回最新位置 | curl 测试 |
| 应用上传 | POST /api/v2/mdm/applications 上传 APK 文件 | curl 测试 |
| 应用列表 | GET /api/v2/mdm/applications 返回应用列表 | curl 测试 |
| 应用安装命令 | POST /api/v2/mdm/devices/:id/install-app 发送 MQTT installApp | 日志确认 |
| 配置创建 | POST /api/v2/mdm/configurations 创建配置 | curl 测试 |
| 配置分配 | POST /api/v2/mdm/configurations/:id/assign 分配到设备 | curl 测试 |

#### Android 端验收

| 功能 | 验收标准 | 测试方法 |
|------|----------|----------|
| 设备信息收集 | DeviceInfoModule.getDeviceInfo() 返回完整设备信息 | Logcat 确认 |
| 设备注册 | 启动时自动注册到 Hub，Hub 收到设备信息 | Hub 日志确认 |
| 心跳发送 | 每 30 秒发送一次心跳，Hub 正确接收 | Hub 日志确认 |
| 位置上报 | GPS 位置变化时自动上报到 Hub | Hub 数据库确认 |
| 命令接收 | MQTT 收到 lock/reboot 命令可正确执行 | 设备行为确认 |
| 应用安装 | 收到 installApp 命令可下载并安装 APK | 设备安装确认 |
| 配置同步 | 拉取并应用 Hub 下发的配置 | 设备设置确认 |

---

### Phase 2 验收标准（第 9-16 周）

#### Hub 端验收

| 功能 | 验收标准 | 测试方法 |
|------|----------|----------|
| 围栏创建 | POST /api/v2/mdm/geofences 创建圆形围栏 | curl 测试 |
| 围栏列表 | GET /api/v2/mdm/geofences 返回租户所有围栏 | curl 测试 |
| 围栏分配 | POST /api/v2/mdm/geofences/:id/devices 分配设备到围栏 | curl 测试 |
| 进入围栏事件 | 设备进入围栏时触发事件通知 | 实际测试 |
| 离开围栏事件 | 设备离开围栏时触发事件通知 | 实际测试 |
| 远程控制会话 | POST /api/v2/mdm/remote-control/sessions 创建会话 | curl 测试 |
| WebRTC 信令 | offer/answer/candidate 正确交换 | 日志确认 |
| 屏幕共享 | Hub 可看到设备屏幕（WebRTC 连接成功） | 实际测试 |
| 消息发送 | POST /api/v2/mdm/messages 发送推送消息 | curl 测试 |
| 消息状态 | 设备收到消息后状态变为 delivered | 数据库确认 |
| 定时消息 | 定时任务按时发送消息 | 实际测试 |
| 用户创建 | POST /api/v2/mdm/users 创建用户 | curl 测试 |
| 角色分配 | PUT /api/v2/mdm/users/:id/role 分配角色 | curl 测试 |
| 权限验证 | 无权限用户访问受保护资源返回 403 | curl 测试 |
| LDAP 配置 | PUT /api/v2/mdm/ldap/config 配置 LDAP | curl 测试 |
| LDAP 同步 | POST /api/v2/mdm/ldap/sync 手动同步用户 | 实际测试 |

#### Android 端验收

| 功能 | 验收标准 | 测试方法 |
|------|----------|----------|
| 围栏检测 | 设备进入/离开围栏触发事件 | 实际测试（移动设备） |
| 围栏事件上报 | 进入/离开围栏时自动上报到 Hub | Hub 日志确认 |
| 会话请求 | 收到远程控制请求时弹出确认对话框 | 设备 UI 确认 |
| 会话接受 | 接受后建立 WebRTC 连接 | Hub 屏幕确认 |
| 屏幕共享 | 设备屏幕实时传输到 Hub | Hub Web UI 确认 |
| 消息接收 | 收到推送消息正确显示通知 | 设备通知确认 |
| 已读回执 | 查看消息后自动发送已读状态 | Hub 数据库确认 |
| 用户登录 | 支持用户名密码登录 | 设备 UI 确认 |
| Token 刷新 | Token 过期前自动刷新 | 日志确认 |

---

## 风险与缓解

### 技术风险

| 风险 | 影响 | 缓解方案 |
|------|------|----------|
| WebRTC 延迟高 | 远程控制体验差 | 使用更高效的编码器，考虑代理模式 |
| MQTT 大规模连接 | 1000+ 设备时性能下降 | MQTT 集群，消息分区 |
| 应用静默安装兼容性 | 部分设备安装失败 | 限制 Android 8+，Device Owner 模式 |
| 地理围栏精度 | GPS 漂移导致误报 | 使用 WiFi 定位辅助，设置合理的围栏半径 |
| 设备端存储限制 | 大量截图/日志占用存储 | 定期清理，云端存储 |

### 安全风险

| 风险 | 影响 | 缓解方案 |
|------|------|----------|
| 设备伪造 | 伪造设备注册到系统 | 设备证书 + JWT 认证 |
| 命令注入 | 中间人篡改 MQTT 命令 | MQTT TLS 加密，命令签名验证 |
| 权限提升 | 普通用户执行管理员操作 | RBAC 严格校验，审计日志 |
| 数据泄露 | 设备位置/信息被窃取 | HTTPS/TLS 加密，数据脱敏 |

---

## 文档版本

| 版本 | 日期 | 作者 | 变更 |
|------|------|------|------|
| 1.0 | 2026-03-24 | Claude | 初始版本 |

