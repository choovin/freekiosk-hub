// Package mqtt_test 提供 MQTT 客户端的集成测试
package mqtt_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/wared2003/freekiosk-hub/internal/models"
	"github.com/wared2003/freekiosk-hub/internal/mqtt"
)

// TestMqttConnection 测试 MQTT 连接
//
// 测试前需要确保 EMQX Broker 正在运行:
//   docker-compose -f deployments/docker-compose.dev.yml up -d emqx
func TestMqttConnection(t *testing.T) {
	// 跳过短测试
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建配置
	config := &mqtt.Config{
		BrokerURL:     "localhost",
		Port:          1883,
		ClientID:      fmt.Sprintf("test-hub-%d", time.Now().Unix()),
		Username:      "",
		Password:      "",
		UseTLS:        false,
		KeepAlive:     30 * time.Second,
		CleanStart:    true,
		AutoReconnect: true,
	}

	// 创建客户端
	client := mqtt.NewClient(config)

	// 连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Disconnect(context.Background())

	// 验证连接状态
	if !client.IsConnected() {
		t.Error("客户端应该已连接")
	}

	t.Log("MQTT 连接测试通过")
}

// TestTopicBuilder 测试 Topic 构建器
func TestTopicBuilder(t *testing.T) {
	tenantID := "tenant001"
	deviceID := "device001"

	builder := mqtt.NewTopicBuilder(tenantID, deviceID)

	// 测试状态 Topic
	expectedStatus := "kiosk/tenant001/device001/status"
	if builder.StatusTopic() != expectedStatus {
		t.Errorf("StatusTopic() = %s, want %s", builder.StatusTopic(), expectedStatus)
	}

	// 测试命令 Topic
	expectedCommand := "kiosk/tenant001/device001/command"
	if builder.CommandTopic() != expectedCommand {
		t.Errorf("CommandTopic() = %s, want %s", builder.CommandTopic(), expectedCommand)
	}

	// 测试配置 Topic
	expectedConfig := "kiosk/tenant001/device001/config"
	if builder.ConfigTopic() != expectedConfig {
		t.Errorf("ConfigTopic() = %s, want %s", builder.ConfigTopic(), expectedConfig)
	}

	// 测试事件 Topic
	expectedEvent := "kiosk/tenant001/device001/event"
	if builder.EventTopic() != expectedEvent {
		t.Errorf("EventTopic() = %s, want %s", builder.EventTopic(), expectedEvent)
	}

	// 测试遥测 Topic
	expectedTelemetry := "kiosk/tenant001/device001/telemetry"
	if builder.TelemetryTopic() != expectedTelemetry {
		t.Errorf("TelemetryTopic() = %s, want %s", builder.TelemetryTopic(), expectedTelemetry)
	}

	// 测试响应 Topic
	commandID := "cmd123"
	expectedResponse := "kiosk/tenant001/device001/response/cmd123"
	if builder.ResponseTopic(commandID) != expectedResponse {
		t.Errorf("ResponseTopic() = %s, want %s", builder.ResponseTopic(commandID), expectedResponse)
	}

	t.Log("Topic 构建器测试通过")
}

// TestDeviceStatusHandler 测试设备状态处理器
func TestDeviceStatusHandler(t *testing.T) {
	// 创建状态通道
	statusChan := make(chan *models.DeviceStatusInfo, 1)

	// 创建处理器
	handler := mqtt.NewDeviceStatusHandler(statusChan)

	// 创建测试状态
	testStatus := models.DeviceStatusInfo{
		DeviceID:         "device001",
		TenantID:         "tenant001",
		UpdatedAt:        time.Now(),
		BatteryLevel:     85,
		BatteryCharging:  true,
		ScreenOn:         true,
		ScreenBrightness: 200,
		Volume:           50,
		WifiSSID:         "TestWiFi",
		WifiStrength:     90,
		IPAddress:        "192.168.1.100",
		CurrentURL:       "https://example.com",
		Loading:          false,
		StorageUsedMB:    1024,
		StorageTotalMB:   16384,
		AppVersion:       "1.0.0",
		Uptime:           3600,
	}

	// 序列化
	payload, err := json.Marshal(testStatus)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	// 处理消息
	if err := handler.Handle("kiosk/tenant001/device001/status", payload); err != nil {
		t.Fatalf("处理失败: %v", err)
	}

	// 接收状态
	select {
	case status := <-statusChan:
		if status.DeviceID != testStatus.DeviceID {
			t.Errorf("DeviceID = %s, want %s", status.DeviceID, testStatus.DeviceID)
		}
		if status.BatteryLevel != testStatus.BatteryLevel {
			t.Errorf("BatteryLevel = %d, want %d", status.BatteryLevel, testStatus.BatteryLevel)
		}
		t.Log("设备状态处理测试通过")
	case <-time.After(1 * time.Second):
		t.Error("未收到状态消息")
	}
}

// TestDeviceEventHandler 测试设备事件处理器
func TestDeviceEventHandler(t *testing.T) {
	// 创建事件通道
	eventChan := make(chan *models.DeviceEvent, 1)

	// 创建处理器
	handler := mqtt.NewDeviceEventHandler(eventChan)

	// 创建测试事件
	testEvent := models.DeviceEvent{
		DeviceID:  "device001",
		TenantID:  "tenant001",
		Type:      "user_interaction",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"action": "tap",
			"x":      100,
			"y":      200,
		},
	}

	// 序列化
	payload, err := json.Marshal(testEvent)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	// 处理消息
	if err := handler.Handle("kiosk/tenant001/device001/event", payload); err != nil {
		t.Fatalf("处理失败: %v", err)
	}

	// 接收事件
	select {
	case event := <-eventChan:
		if event.DeviceID != testEvent.DeviceID {
			t.Errorf("DeviceID = %s, want %s", event.DeviceID, testEvent.DeviceID)
		}
		if event.Type != testEvent.Type {
			t.Errorf("Type = %s, want %s", event.Type, testEvent.Type)
		}
		t.Log("设备事件处理测试通过")
	case <-time.After(1 * time.Second):
		t.Error("未收到事件消息")
	}
}

// TestDeviceTelemetryHandler 测试设备遥测处理器
func TestDeviceTelemetryHandler(t *testing.T) {
	// 创建遥测通道
	telemetryChan := make(chan *models.DeviceTelemetry, 1)

	// 创建处理器
	handler := mqtt.NewDeviceTelemetryHandler(telemetryChan)

	// 创建测试遥测
	testTelemetry := models.DeviceTelemetry{
		DeviceID:     "device001",
		TenantID:     "tenant001",
		Timestamp:    time.Now(),
		CPUUsage:     45.5,
		MemoryUsage:  60.2,
		NetworkTX:    1024000,
		NetworkRX:    2048000,
		Temperature:  35.5,
	}

	// 序列化
	payload, err := json.Marshal(testTelemetry)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	// 处理消息
	if err := handler.Handle("kiosk/tenant001/device001/telemetry", payload); err != nil {
		t.Fatalf("处理失败: %v", err)
	}

	// 接收遥测
	select {
	case telemetry := <-telemetryChan:
		if telemetry.DeviceID != testTelemetry.DeviceID {
			t.Errorf("DeviceID = %s, want %s", telemetry.DeviceID, testTelemetry.DeviceID)
		}
		if telemetry.CPUUsage != testTelemetry.CPUUsage {
			t.Errorf("CPUUsage = %f, want %f", telemetry.CPUUsage, testTelemetry.CPUUsage)
		}
		t.Log("设备遥测处理测试通过")
	case <-time.After(1 * time.Second):
		t.Error("未收到遥测消息")
	}
}

// TestCommandResponseHandler 测试命令响应处理器
func TestCommandResponseHandler(t *testing.T) {
	// 创建处理器
	handler := mqtt.NewCommandResponseHandler()

	// 创建响应通道
	responseChan := make(chan *models.CommandResult, 1)
	commandID := "cmd-12345"

	// 注册通道
	handler.Register(commandID, responseChan)

	// 创建测试响应
	testResult := models.CommandResult{
		CommandID: commandID,
		Success:   true,
		Result:    map[string]interface{}{"message": "操作成功"},
		Timestamp: time.Now(),
	}

	// 序列化
	payload, err := json.Marshal(testResult)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	// 处理消息
	if err := handler.Handle("kiosk/tenant001/device001/response/cmd-12345", payload); err != nil {
		t.Fatalf("处理失败: %v", err)
	}

	// 接收响应
	select {
	case result := <-responseChan:
		if result.CommandID != testResult.CommandID {
			t.Errorf("CommandID = %s, want %s", result.CommandID, testResult.CommandID)
		}
		if result.Success != testResult.Success {
			t.Errorf("Success = %v, want %v", result.Success, testResult.Success)
		}
		t.Log("命令响应处理测试通过")
	case <-time.After(1 * time.Second):
		t.Error("未收到命令响应")
	}

	// 注销通道
	handler.Unregister(commandID)
}

// TestMqttPublishSubscribe 测试 MQTT 发布和订阅
//
// 这是一个完整的集成测试，需要运行 EMQX Broker
func TestMqttPublishSubscribe(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 创建配置
	config := &mqtt.Config{
		BrokerURL:     "localhost",
		Port:          1883,
		ClientID:      fmt.Sprintf("test-pubsub-%d", time.Now().Unix()),
		Username:      "",
		Password:      "",
		UseTLS:        false,
		KeepAlive:     30 * time.Second,
		CleanStart:    true,
		AutoReconnect: true,
	}

	// 创建客户端
	client := mqtt.NewClient(config)

	// 连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer client.Disconnect(context.Background())

	// 创建消息通道
	messageChan := make(chan []byte, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	// 订阅测试 Topic
	testTopic := "test/integration/message"
	if err := client.Subscribe(ctx, testTopic, func(topic string, payload []byte) error {
		messageChan <- payload
		wg.Done()
		return nil
	}); err != nil {
		t.Fatalf("订阅失败: %v", err)
	}

	// 等待订阅生效
	time.Sleep(500 * time.Millisecond)

	// 发布消息
	testMessage := []byte(`{"test": "hello world"}`)
	if err := client.Publish(ctx, testTopic, testMessage); err != nil {
		t.Fatalf("发布失败: %v", err)
	}

	// 等待接收
	select {
	case received := <-messageChan:
		if string(received) != string(testMessage) {
			t.Errorf("收到消息 = %s, want %s", string(received), string(testMessage))
		}
		t.Log("发布订阅测试通过")
	case <-time.After(5 * time.Second):
		t.Error("未收到发布的消息")
	}
}

// TestConfigFromEnv 测试从环境变量创建配置
func TestConfigFromEnv(t *testing.T) {
	// 注意：这个测试使用默认值，因为没有设置环境变量
	config := mqtt.ConfigFromEnv()

	if config.BrokerURL != "localhost" {
		t.Errorf("BrokerURL = %s, want localhost", config.BrokerURL)
	}
	if config.Port != 1883 {
		t.Errorf("Port = %d, want 1883", config.Port)
	}
	if config.ClientID != "freekiosk-hub" {
		t.Errorf("ClientID = %s, want freekiosk-hub", config.ClientID)
	}
	if config.KeepAlive != 60*time.Second {
		t.Errorf("KeepAlive = %v, want 60s", config.KeepAlive)
	}

	t.Log("环境变量配置测试通过")
}

// TestSharedSubscription 测试共享订阅
func TestSharedSubscription(t *testing.T) {
	group := "hub-cluster"
	expected := "$share/hub-cluster/kiosk/+/+/command"

	result := mqtt.SharedCommandSubscription(group)
	if result != expected {
		t.Errorf("SharedCommandSubscription() = %s, want %s", result, expected)
	}

	t.Log("共享订阅测试通过")
}

// TestWildcardTopics 测试通配符 Topic
func TestWildcardTopics(t *testing.T) {
	tenantID := "tenant001"

	// 测试状态通配符
	statusWildcard := mqtt.StatusWildcard(tenantID)
	expectedStatus := "kiosk/tenant001/+/status"
	if statusWildcard != expectedStatus {
		t.Errorf("StatusWildcard() = %s, want %s", statusWildcard, expectedStatus)
	}

	// 测试事件通配符
	eventWildcard := mqtt.EventWildcard(tenantID)
	expectedEvent := "kiosk/tenant001/+/event"
	if eventWildcard != expectedEvent {
		t.Errorf("EventWildcard() = %s, want %s", eventWildcard, expectedEvent)
	}

	// 测试遥测通配符
	telemetryWildcard := mqtt.TelemetryWildcard(tenantID)
	expectedTelemetry := "kiosk/tenant001/+/telemetry"
	if telemetryWildcard != expectedTelemetry {
		t.Errorf("TelemetryWildcard() = %s, want %s", telemetryWildcard, expectedTelemetry)
	}

	t.Log("通配符 Topic 测试通过")
}