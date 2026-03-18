// Package mqtt 提供 MQTT 5.0 客户端功能
//
// 用于 freekiosk-hub 与 EMQX Broker 之间的通信，
// 实现设备状态监控、命令下发和配置同步。
package mqtt

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config MQTT 连接配置
type Config struct {
	// Broker 地址
	BrokerURL string
	// Broker 端口
	Port int
	// 客户端 ID
	ClientID string
	// 认证用户名
	Username string
	// 认证密码
	Password string
	// 是否使用 TLS
	UseTLS bool
	// 心跳间隔
	KeepAlive time.Duration
	// 是否清除会话
	CleanStart bool
	// 是否自动重连
	AutoReconnect bool
}

// ConfigFromEnv 从环境变量创建配置
//
// 环境变量:
//   - MQTT_BROKER_URL: Broker 地址 (默认: localhost)
//   - MQTT_PORT: Broker 端口 (默认: 1883)
//   - MQTT_CLIENT_ID: 客户端 ID (默认: freekiosk-hub)
//   - MQTT_USERNAME: 认证用户名
//   - MQTT_PASSWORD: 认证密码
//   - MQTT_USE_TLS: 是否使用 TLS (默认: false)
func ConfigFromEnv() *Config {
	return &Config{
		BrokerURL:     getEnv("MQTT_BROKER_URL", "localhost"),
		Port:          getEnvInt("MQTT_PORT", 1883),
		ClientID:      getEnv("MQTT_CLIENT_ID", "freekiosk-hub"),
		Username:      os.Getenv("MQTT_USERNAME"),
		Password:      os.Getenv("MQTT_PASSWORD"),
		UseTLS:        getEnvBool("MQTT_USE_TLS", false),
		KeepAlive:     60 * time.Second,
		CleanStart:    false,
		AutoReconnect: true,
	}
}

// getEnv 获取环境变量，支持默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取整数类型环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBool 获取布尔类型环境变量
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}

// BrokerAddress 返回完整的 Broker 地址
func (c *Config) BrokerAddress() string {
	return fmt.Sprintf("%s:%d", c.BrokerURL, c.Port)
}