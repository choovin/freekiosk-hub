# FreeKiosk MQTT 集成技术方案

> 文档版本：1.0
> 创建日期：2026-03-15
> 状态：设计方案

---

## 一、架构设计

### 1.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        公网 / 云环境                              │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    EMQX 消息服务器                         │   │
│  │                   (yanxue-emqx:6.1.1)                    │   │
│  │  ┌─────────────────────────────────────────────────────┐ │   │
│  │  │  Topic: freekiosk/+/command  (设备命令)              │ │   │
│  │  │  Topic: freekiosk/+/status   (设备状态)              │ │   │
│  │  │  Topic: freekiosk/+/online   (在线状态)              │ │   │
│  │  └─────────────────────────────────────────────────────┘ │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              ↑                                   │
│              ┌───────────────┼───────────────┐                  │
│              │               │               │                  │
│  ┌───────────┴────┐  ┌───────┴───────┐  ┌───┴──────────────┐   │
│  │  FreeKiosk Hub │  │  FreeKiosk #1 │  │  FreeKiosk #N    │   │
│  │   (Go 服务端)   │  │  (Android)    │  │  (Android)       │   │
│  │  - 发布命令     │  │  - 订阅命令    │  │  - 订阅命令       │   │
│  │  - 订阅状态     │  │  - 发布状态    │  │  - 发布状态       │   │
│  └─────────────────┘  └───────────────┘  └──────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 通信模式

| 方向 | Topic | 发布者 | 订阅者 | 用途 | QoS |
|------|-------|--------|--------|------|-----|
| 下行 | `freekiosk/{device_id}/command` | Hub | 所有设备 | 发送控制命令 | 1 |
| 下行 | `freekiosk/{device_id}/config` | Hub | 所有设备 | 推送配置更新 | 1 |
| 上行 | `freekiosk/{device_id}/status` | 设备 | Hub | 上报设备状态 | 1 |
| 上行 | `freekiosk/{device_id}/online` | 设备 | Hub | 在线/离线通知 (LWT) | 1 |
| 上行 | `freekiosk/{device_id}/response` | 设备 | Hub | 命令执行响应 | 1 |

---

## 二、EMQX 服务器配置

### 2.1 Docker Compose 配置

```yaml
version: '3.8'

services:
  # EMQX MQTT 消息服务器
  emqx:
    image: docker.1ms.run/emqx/emqx:6.1.1
    container_name: yanxue-emqx
    restart: unless-stopped
    environment:
      - EMQX_NAME=yanxue
      - EMQX_HOST=0.0.0.0
      # 认证配置（可选，建议生产环境启用）
      - EMQX_AUTH__USER__1=freekiosk
      - EMQX_AUTH__PASSWORD__1=YourSecurePassword123
    ports:
      - "1883:1883"    # MQTT TCP
      - "8883:8883"    # MQTT TLS（如需加密）
      - "8083:8083"    # WebSocket
      - "8084:8084"    # WebSocket SSL
      - "18083:18083"  # Dashboard 管理界面
    volumes:
      - emqx_data:/opt/emqx/data
      - emqx_log:/opt/emqx/log
      # 可选：挂载自定义配置文件
      - ./emqx.conf:/opt/emqx/etc/emqx.conf:ro
    networks:
      - freekiosk-net

  # FreeKiosk Hub 服务端
  freekiosk-hub:
    build: ./freekiosk-hub
    container_name: freekiosk-hub
    restart: unless-stopped
    environment:
      - SERVER_PORT=8081
      - MQTT_BROKER=tcp://yanxue-emqx:1883
      - MQTT_USERNAME=freekiosk
      - MQTT_PASSWORD=YourSecurePassword123
      - MQTT_CLIENT_ID=freekiosk-hub-server
      - USE_MQTT=true
    ports:
      - "8081:8081"
    volumes:
      - hub_data:/app/data
    depends_on:
      - emqx
    networks:
      - freekiosk-net

volumes:
  emqx_data:
  emqx_log:
  hub_data:

networks:
  freekiosk-net:
    driver: bridge
```

### 2.2 EMQX 访问信息

| 服务 | 地址 | 说明 |
|------|------|------|
| MQTT Broker | `tcp://your-server-ip:1883` | 设备连接地址 |
| MQTT WebSocket | `ws://your-server-ip:8083/mqtt` | WebSocket 连接 |
| Dashboard | `http://your-server-ip:18083` | 管理后台 |
| Dashboard 默认账号 | `admin` / `public` | 建议修改 |

---

## 三、服务端 (freekiosk-hub) 实现

### 3.1 目录结构

```
freekiosk-hub/
├── internal/
│   ├── mqtt/
│   │   ├── client.go          # MQTT 客户端封装
│   │   ├── topics.go          # Topic 常量定义
│   │   └── messages.go        # 消息结构定义
│   ├── config/
│   │   └── config.go          # 配置管理（新增 MQTT 配置）
│   └── ...
├── cmd/server/main.go         # 主入口（新增 MQTT 初始化）
└── go.mod                     # 新增 paho.mqtt.golang 依赖
```

### 3.2 代码实现

#### 3.2.1 Topic 定义 (`internal/mqtt/topics.go`)

```go
package mqtt

import "fmt"

// Topic 模板
const (
	// 下行主题（Hub 发布，设备订阅）
	TopicCommandTemplate = "freekiosk/%s/command"   // 控制命令
	TopicConfigTemplate  = "freekiosk/%s/config"    // 配置更新

	// 上行主题（设备发布，Hub 订阅）
	TopicStatusTemplate   = "freekiosk/%s/status"   // 设备状态
	TopicOnlineTemplate   = "freekiosk/%s/online"   // 在线状态
	TopicResponseTemplate = "freekiosk/%s/response" // 命令响应
)

// 通配符订阅（用于接收所有设备消息）
const (
	WildcardStatus   = "freekiosk/+/status"
	WildcardOnline   = "freekiosk/+/online"
	WildcardResponse = "freekiosk/+/response"
)

// BuildCommandTopic 构建设备命令主题
func BuildCommandTopic(deviceID string) string {
	return fmt.Sprintf(TopicCommandTemplate, deviceID)
}

// BuildStatusTopic 构建状态主题
func BuildStatusTopic(deviceID string) string {
	return fmt.Sprintf(TopicStatusTemplate, deviceID)
}

// ExtractDeviceID 从主题中提取设备 ID
func ExtractDeviceID(topic string) string {
	// freekiosk/xxx/status -> xxx
	parts := splitTopic(topic)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

func splitTopic(topic string) []string {
	result := []string{}
	current := ""
	for _, c := range topic {
		if c == '/' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	result = append(result, current)
	return result
}
```

#### 3.2.2 消息结构 (`internal/mqtt/messages.go`)

```go
package mqtt

import "time"

// CommandMessage Hub 发送给设备的命令
type CommandMessage struct {
	MessageID string                 `json:"messageId"`
	Action    string                 `json:"action"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Timeout   int                    `json:"timeout,omitempty"` // 超时时间 (秒)
}

// ResponseMessage 设备返回的命令响应
type ResponseMessage struct {
	MessageID   string      `json:"messageId"`
	DeviceID    string      `json:"deviceId"`
	Success     bool        `json:"success"`
	Action      string      `json:"action"`
	Data        interface{} `json:"data,omitempty"`
	Error       string      `json:"error,omitempty"`
	Timestamp   int64       `json:"timestamp"`
}

// StatusMessage 设备上报的状态
type StatusMessage struct {
	DeviceID  string       `json:"deviceId"`
	Timestamp int64        `json:"timestamp"`
	Data      DeviceStatus `json:"data"`
}

// DeviceStatus 设备状态数据结构
type DeviceStatus struct {
	Battery    BatteryInfo    `json:"battery"`
	Screen     ScreenInfo     `json:"screen"`
	Audio      AudioInfo      `json:"audio"`
	Webview    WebviewInfo    `json:"webview"`
	Device     DeviceInfo     `json:"device"`
	Wifi       WifiInfo       `json:"wifi"`
	Connection ConnectionInfo `json:"connection"`
}

type BatteryInfo struct {
	Level    int    `json:"level"`
	Charging bool   `json:"charging"`
	Plugged  string `json:"plugged"`
}

type ScreenInfo struct {
	On                bool `json:"on"`
	Brightness        int  `json:"brightness"`
	ScreensaverActive bool `json:"screensaverActive"`
}

type AudioInfo struct {
	Volume int `json:"volume"`
}

type WebviewInfo struct {
	CurrentUrl string `json:"currentUrl"`
	CanGoBack  bool   `json:"canGoBack"`
	Loading    bool   `json:"loading"`
}

type DeviceInfo struct {
	IP            string `json:"ip"`
	Hostname      string `json:"hostname"`
	Version       string `json:"version"`
	IsDeviceOwner bool   `json:"isDeviceOwner"`
	KioskMode     bool   `json:"kioskMode"`
}

type WifiInfo struct {
	SSID           string `json:"ssid"`
	SignalStrength int    `json:"signalStrength"`
	SignalLevel    int    `json:"signalLevel"`
	Connected      bool   `json:"connected"`
	LinkSpeed      int    `json:"linkSpeed"`
	Frequency      int    `json:"frequency"`
}

type ConnectionInfo struct {
	LastSeen   time.Time `json:"lastSeen"`
	Connection string    `json:"connection"` // "mqtt" or "http"
}

// OnlineMessage 在线/离线通知
type OnlineMessage struct {
	DeviceID  string `json:"deviceId"`
	Timestamp int64  `json:"timestamp"`
	Online    bool   `json:"online"`
}
```

#### 3.2.3 MQTT 客户端 (`internal/mqtt/client.go`)

```go
package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Config MQTT 配置
type Config struct {
	Broker   string
	Username string
	Password string
	ClientID string
}

// Client MQTT 客户端
type Client struct {
	client      mqtt.Client
	config      Config
	msgHandlers map[string]func(string, *StatusMessage)
	respHandlers map[string]chan *ResponseMessage
	mu          sync.RWMutex
}

// NewClient 创建 MQTT 客户端
func NewClient(cfg Config) *Client {
	return &Client{
		config:      cfg,
		msgHandlers: make(map[string]func(string, *StatusMessage)),
		respHandlers: make(map[string]chan *ResponseMessage),
	}
}

// Connect 连接到 MQTT Broker
func (c *Client) Connect(ctx context.Context) error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(c.config.Broker)
	opts.SetClientID(c.config.ClientID)
	opts.SetUsername(c.config.Username)
	opts.SetPassword(c.config.Password)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetMaxReconnectInterval(60 * time.Second)
	opts.SetCleanSession(false) // 持久会话

	// 连接成功回调
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		slog.Info("✅ MQTT connected to broker", "broker", c.config.Broker)
		c.subscribeToTopics(client)
	})

	// 连接断开回调
	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		slog.Warn("⚠️ MQTT connection lost", "error", err)
	})

	// 创建客户端
	c.client = mqtt.NewClient(opts)

	// 连接
	token := c.client.Connect()
	token.Wait()
	if token.Error() != nil {
		return fmt.Errorf("failed to connect MQTT: %w", token.Error())
	}

	slog.Info("📡 MQTT client initialized")
	return nil
}

// subscribeToTopics 订阅主题
func (c *Client) subscribeToTopics(client mqtt.Client) {
	topics := []struct {
		topic   string
		qos     byte
		handler mqtt.MessageHandler
	}{
		{WildcardStatus, 1, c.handleStatusMessage},
		{WildcardOnline, 1, c.handleOnlineMessage},
		{WildcardResponse, 1, c.handleResponseMessage},
	}

	for _, t := range topics {
		token := client.Subscribe(t.topic, t.qos, t.handler)
		token.Wait()
		if token.Error() != nil {
			slog.Error("Failed to subscribe topic", "topic", t.topic, "error", token.Error())
		} else {
			slog.Info("📬 Subscribed to topic", "topic", t.topic)
		}
	}
}

// 消息处理函数
func (c *Client) handleStatusMessage(client mqtt.Client, msg mqtt.Message) {
	var status StatusMessage
	if err := json.Unmarshal(msg.Payload(), &status); err != nil {
		slog.Error("Failed to parse status message", "error", err)
		return
	}

	c.mu.RLock()
	handler := c.msgHandlers[status.DeviceID]
	c.mu.RUnlock()

	if handler != nil {
		handler(status.DeviceID, &status)
	} else {
		// 默认处理：记录日志
		slog.Debug("Received status", "device", status.DeviceID, "data", status.Data)
	}
}

func (c *Client) handleOnlineMessage(client mqtt.Client, msg mqtt.Message) {
	var online OnlineMessage
	if err := json.Unmarshal(msg.Payload(), &online); err != nil {
		slog.Error("Failed to parse online message", "error", err)
		return
	}

	status := "offline"
	if online.Online {
		status = "online"
	}
	slog.Info("Device status changed", "device", online.DeviceID, "status", status)
}

func (c *Client) handleResponseMessage(client mqtt.Client, msg mqtt.Message) {
	var resp ResponseMessage
	if err := json.Unmarshal(msg.Payload(), &resp); err != nil {
		slog.Error("Failed to parse response message", "error", err)
		return
	}

	c.mu.RLock()
	ch := c.respHandlers[resp.MessageID]
	c.mu.RUnlock()

	if ch != nil {
		select {
		case ch <- &resp:
		default:
			// 通道已满，丢弃
		}
	}
}

// RegisterStatusHandler 注册状态处理器
func (c *Client) RegisterStatusHandler(deviceID string, handler func(string, *StatusMessage)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgHandlers[deviceID] = handler
}

// PublishCommand 发送命令到设备
func (c *Client) PublishCommand(ctx context.Context, deviceID, action string, params map[string]interface{}, timeout time.Duration) (*ResponseMessage, error) {
	messageID := fmt.Sprintf("%s-%d", deviceID, time.Now().UnixNano())

	cmd := CommandMessage{
		MessageID: messageID,
		Action:    action,
		Params:    params,
		Timestamp: time.Now().UnixMilli(),
		Timeout:   int(timeout.Seconds()),
	}

	payload, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	// 创建响应通道
	respCh := make(chan *ResponseMessage, 1)
	c.mu.Lock()
	c.respHandlers[messageID] = respCh
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.respHandlers, messageID)
		c.mu.Unlock()
		close(respCh)
	}()

	// 发布命令
	topic := BuildCommandTopic(deviceID)
	token := c.client.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		return nil, token.Error()
	}

	slog.Debug("Command published", "device", deviceID, "action", action, "topic", topic)

	// 等待响应
	select {
	case resp := <-respCh:
		return resp, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("command timeout")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// PublishConfig 发送配置到设备
func (c *Client) PublishConfig(ctx context.Context, deviceID string, config map[string]interface{}) error {
	payload, err := json.Marshal(config)
	if err != nil {
		return err
	}

	topic := fmt.Sprintf(TopicConfigTemplate, deviceID)
	token := c.client.Publish(topic, 1, false, payload)
	token.Wait()
	return token.Error()
}

// Disconnect 断开连接
func (c *Client) Disconnect() {
	if c.client != nil {
		c.client.Disconnect(1000)
	}
}
```

### 3.3 配置更新 (`internal/config/config.go`)

```go
type Config struct {
	// 现有配置
	ServerPort     string `env:"SERVER_PORT" default:"8081"`
	DBPath         string `env:"DB_PATH" default:"freekiosk.db"`
	KioskPort      string `env:"KIOSK_PORT" default:"8080"`
	KioskApiKey    string `env:"KIOSK_API_KEY"`
	PollInterval   time.Duration
	RetentionDays  int
	MaxWorkers     int

	// MQTT 配置（新增）
	MQTTBroker   string `env:"MQTT_BROKER" default:"tcp://localhost:1883"`
	MQTTUsername string `env:"MQTT_USERNAME" default:""`
	MQTTPassword string `env:"MQTT_PASSWORD" default:""`
	MQTTClientID string `env:"MQTT_CLIENT_ID" default:"freekiosk-hub"`
	UseMQTT      bool   `env:"USE_MQTT" default:"false"`
}
```

### 3.4 主程序更新 (`cmd/server/main.go`)

在 `main.go` 中添加 MQTT 初始化逻辑（第 64-89 行附近）：

```go
var httpClient *http.Client
var mqttClient *mqtt.Client

// 2. Network Management (Tailscale vs MQTT vs Standard)
if cfg.UseMQTT {
	slog.Info("📡 Initializing MQTT client...")
	mqttCfg := mqtt.Config{
		Broker:   cfg.MQTTBroker,
		Username: cfg.MQTTUsername,
		Password: cfg.MQTTPassword,
		ClientID: cfg.MQTTClientID,
	}
	mqttClient = mqtt.NewClient(mqttCfg)

	if err := mqttClient.Connect(ctx); err != nil {
		slog.Error("Failed to connect MQTT", "error", err)
		// 可选择是否退出或降级到 HTTP 模式
	}
} else if cfg.TSAuthKey != "" {
	slog.Info("🔐 Tailscale auth key detected, connecting to tailnet...")
	// ... 现有 Tailscale 代码
} else {
	slog.Warn("⚠️ No Tailscale key found. Using standard network stack.")
	httpClient = &http.Client{
		Timeout: 15 * time.Second,
	}
}
```

---

## 四、Android 客户端 (freekiosk) 集成

### 4.1 配置界面新增

在设置界面添加 MQTT 配置选项：

```typescript
// src/screens/settings/tabs/NetworkTab.tsx
interface MQTTConfig {
  enabled: boolean;
  broker: string;      // tcp://broker.emqx.io:1883
  username: string;
  password: string;
  clientId: string;
  topicPrefix: string; // freekiosk
}
```

### 4.2 Kotlin MQTT 客户端增强

在现有 `KioskMqttClient.kt` 基础上，添加对 Hub 命令主题的订阅：

```kotlin
// android/app/src/main/java/com/freekiosk/mqtt/KioskMqttClient.kt

class KioskMqttClient {
    // ... 现有代码 ...

    private val commandTopic = "freekiosk/$deviceId/command"
    private val statusTopic = "freekiosk/$deviceId/status"
    private val onlineTopic = "freekiosk/$deviceId/online"

    fun connect(config: MQTTConfig) {
        // 订阅命令主题
        mqttClient.subscribe(commandTopic, 1) { topic, message ->
            handleCommand(message.toString(Charsets.UTF_8))
        }

        // 发布在线状态
        publishOnlineStatus(true)
    }

    private fun handleCommand(json: String) {
        val cmd = parseCommandMessage(json)
        when (cmd.action) {
            "setBrightness" -> setBrightness(cmd.params["value"] as Int)
            "navigate" -> navigate(cmd.params["url"] as String)
            "reboot" -> reboot()
            // ... 其他命令
        }

        // 发送响应
        publishResponse(cmd.messageId, true, null)
    }

    private fun publishResponse(messageId: String, success: Boolean, data: Any?) {
        val response = ResponseMessage(
            messageId = messageId,
            deviceId = deviceId,
            success = success,
            action = action,
            data = data,
            timestamp = System.currentTimeMillis()
        )
        mqttClient.publish("freekiosk/$deviceId/response", response.toJson())
    }
}
```

---

## 五、环境变量配置

### 5.1 服务端 .env.example

```dotenv
# FreeKiosk Hub Configuration
# Copy this file to .env and fill in your values

# -- Server Configuration --
SERVER_PORT=8081
LOG_LEVEL=INFO

# -- Database --
DB_PATH=/app/data/freekiosk.db

# -- Kiosk Communication --
KIOSK_PORT=8080
KIOSK_API_KEY=your-secret-api-key-here

# -- MQTT Integration (Optional) --
# Enable MQTT for public network communication
USE_MQTT=false
MQTT_BROKER=tcp://your-emqx-server.com:1883
MQTT_USERNAME=freekiosk
MQTT_PASSWORD=YourSecurePassword123
MQTT_CLIENT_ID=freekiosk-hub-server

# -- Tailscale Integration (Alternative) --
# Create an API key from the Tailscale admin console:
# https://login.tailscale.com/admin/settings/keys
TS_AUTHKEY=

# -- Performance --
POLL_INTERVAL=30s
RETENTION_DAYS=31
MAX_WORKERS=5

# -- Media --
BASE_URL=http://localhost:8081
```

### 5.2 客户端配置（通过 Hub 推送或手动设置）

```
MQTT Broker: tcp://your-emqx-server.com:1883
Username: freekiosk-device-001
Password: DevicePassword123
Client ID: freekiosk-001
Topic Prefix: freekiosk
```

---

## 六、安全建议

### 6.1 EMQX 认证配置

```bash
# 为每个设备创建独立账号
EMQX_AUTH__USER__1=freekiosk-hub
EMQX_AUTH__PASSWORD__1=HubPassword123

EMQX_AUTH__USER__2=freekiosk-device-001
EMQX_AUTH__PASSWORD__2=Device1Password

EMQX_AUTH__USER__3=freekiosk-device-002
EMQX_AUTH__PASSWORD__3=Device2Password
```

### 6.2 TLS 加密（推荐生产环境）

```yaml
ports:
  - "8883:8883"  # MQTT TLS

volumes:
  - ./certs/server.crt:/opt/emqx/etc/certs/server.crt
  - ./certs/server.key:/opt/emqx/etc/certs/server.key
```

```dotenv
MQTT_BROKER=ssl://your-emqx-server.com:8883
```

### 6.3 ACL 访问控制

在 EMQX Dashboard 配置 ACL 规则：

```
# Hub 可以发布到所有 command 主题，订阅所有 status 主题
{allow, {user, "freekiosk-hub"}, publish, ["freekiosk/+/command", "freekiosk/+/config"]}.
{allow, {user, "freekiosk-hub"}, subscribe, ["freekiosk/+/status", "freekiosk/+/online", "freekiosk/+/response"]}.

# 设备只能订阅自己的 command 主题，发布自己的 status 主题
{allow, {user, "freekiosk-device-001"}, publish, ["freekiosk/device-001/+", "freekiosk/device-001/online"]}.
{allow, {user, "freekiosk-device-001"}, subscribe, ["freekiosk/device-001/command", "freekiosk/device-001/config"]}.
```

---

## 七、实施步骤

### 阶段一：环境搭建（1-2 天）

1. 部署 EMQX 服务器
2. 配置认证和 ACL
3. 测试 MQTT 连接（使用 MQTTX 等工具）

### 阶段二：服务端开发（3-5 天）

1. 实现 MQTT 客户端模块
2. 实现消息编解码
3. 实现命令 - 响应机制
4. 集成到现有 Monitor 服务

### 阶段三：客户端开发（2-3 天）

1. 添加 MQTT 配置界面
2. 增强 MQTT 命令处理
3. 实现状态上报
4. 实现离线消息缓存

### 阶段四：测试与优化（2-3 天）

1. 功能测试
2. 压力测试（多设备并发）
3. 断线重连测试
4. 消息可靠性验证

---

## 八、消息格式示例

### 8.1 亮度调节命令

```json
// Hub → 设备 (freekiosk/device-001/command)
{
  "messageId": "device-001-1710512345678",
  "action": "setBrightness",
  "params": {"value": 80},
  "timestamp": 1710512345678,
  "timeout": 10
}

// 设备 → Hub (freekiosk/device-001/response)
{
  "messageId": "device-001-1710512345678",
  "deviceId": "device-001",
  "success": true,
  "action": "setBrightness",
  "data": {"brightness": 80},
  "timestamp": 1710512345688
}
```

### 8.2 状态上报

```json
// 设备 → Hub (freekiosk/device-001/status)
{
  "deviceId": "device-001",
  "timestamp": 1710512400000,
  "data": {
    "battery": {"level": 85, "charging": true, "plugged": "ac"},
    "screen": {"on": true, "brightness": 80, "screensaverActive": false},
    "audio": {"volume": 50},
    "webview": {"currentUrl": "http://example.com", "canGoBack": false, "loading": false},
    "device": {"ip": "192.168.1.100", "hostname": "kiosk-001", "version": "1.2.0", "isDeviceOwner": true, "kioskMode": true},
    "wifi": {"ssid": "MyWiFi", "signalStrength": -65, "signalLevel": 4, "connected": true, "linkSpeed": 72, "frequency": 2437},
    "connection": {"lastSeen": "2024-03-15T10:00:00Z", "connection": "mqtt"}
  }
}
```

### 8.3 在线/离线通知（LWT）

```json
// 设备上线 (freekiosk/device-001/online)
{
  "deviceId": "device-001",
  "timestamp": 1710512400000,
  "online": true
}

// 设备离线（Last Will Testament）
{
  "deviceId": "device-001",
  "timestamp": 1710512500000,
  "online": false
}
```

---

## 九、故障排查

| 问题 | 可能原因 | 解决方案 |
|------|----------|----------|
| 设备无法连接 | Broker 地址错误 | 检查网络和设备配置 |
| 命令无响应 | 主题不匹配 | 检查 deviceID 是否正确 |
| 消息丢失 | QoS 设置过低 | 使用 QoS 1 或 2 |
| 频繁断线 | 心跳超时 | 调整 keepalive 参数 |
| 响应超时 | 设备离线或网络差 | 增加 timeout，实现重试机制 |

---

## 十、依赖添加

### 10.1 Go 服务端 (`go.mod`)

```go
require github.com/eclipse/paho.mqtt.golang v1.5.0
```

### 10.2 Android 客户端 (`build.gradle`)

现有 MQTT 依赖已包含（HiveMQ），无需额外添加。

---

## 十一、混合模式支持

系统支持三种通信模式，可共存：

| 模式 | 优先级 | 说明 |
|------|--------|------|
| MQTT | 高 | 公网设备，首选模式 |
| Tailscale | 中 | 内网/虚拟局域网设备 |
| HTTP 直连 | 低 | 局域网设备，降级方案 |

在 `main.go` 中按优先级初始化：

```go
if cfg.UseMQTT {
    // 初始化 MQTT
} else if cfg.TSAuthKey != "" {
    // 初始化 Tailscale
} else {
    // 使用标准 HTTP
}
```

---

## 附录

### A. 测试工具

- [MQTTX](https://mqttx.app/) - 跨平台 MQTT 客户端
- [MQTT Explorer](http://mqtt-explorer.com/) - MQTT 调试工具
- EMQX Dashboard - `http://localhost:18083`

### B. 相关文档

- [EMQX 官方文档](https://www.emqx.com/docs/)
- [Paho MQTT Go 客户端](https://github.com/eclipse/paho.mqtt.golang)
- [freekiosk MQTT 文档](./freekiosk/docs/MQTT.md)
