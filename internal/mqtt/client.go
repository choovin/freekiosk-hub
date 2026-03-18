package mqtt

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.golang/autopaho"
	mqttv5 "github.com/eclipse/paho.golang/paho"
)

// Client MQTT 5.0 客户端封装
//
// 提供与 EMQX Broker 的双向通信能力，支持:
// - 设备状态接收
// - 命令下发
// - 共享订阅（负载均衡）
// - 保留消息
type Client struct {
	config     *Config
	connection *mqtt.ConnectionManager
	handlers   map[string]MessageHandler
	router     *mqttv5.StandardRouter
	mu         sync.RWMutex
}

// MessageHandler 消息处理函数类型
type MessageHandler func(topic string, payload []byte) error

// NewClient 创建 MQTT 客户端
func NewClient(config *Config) *Client {
	return &Client{
		config:   config,
		handlers: make(map[string]MessageHandler),
	}
}

// Connect 连接到 MQTT Broker
func (c *Client) Connect(ctx context.Context) error {
	brokerURL := fmt.Sprintf("tcp://%s:%d", c.config.BrokerURL, c.config.Port)

	// 创建消息路由器，设置默认处理器
	c.router = mqttv5.NewStandardRouterWithDefault(func(p *mqttv5.Publish) {
		c.handleMessage(p.Topic, p.Payload)
	})

	// 配置客户端
	// 注意: autopaho.ClientConfig 内嵌了 paho.ClientConfig
	clientConfig := mqtt.ClientConfig{
		ServerUrls:                    []*url.URL{mustParseURL(brokerURL)},
		KeepAlive:                     uint16(c.config.KeepAlive.Seconds()),
		CleanStartOnInitialConnection: c.config.CleanStart,
		SessionExpiryInterval:         uint32(3600), // 1小时会话过期
		ConnectTimeout:                10 * time.Second,
		OnConnectionUp: func(cm *mqtt.ConnectionManager, connAck *mqttv5.Connack) {
			log.Printf("[MQTT] 已连接到 %s", brokerURL)
		},
		OnConnectionDown: func() bool {
			log.Printf("[MQTT] 连接断开，尝试重连...")
			return true // 返回 true 继续重连
		},
		OnConnectError: func(err error) {
			log.Printf("[MQTT] 连接错误: %v", err)
		},
	}

	// 配置 paho 客户端选项 (内嵌在 ClientConfig 中)
	clientConfig.ClientID = c.config.ClientID
	clientConfig.Router = c.router

	// 配置认证
	if c.config.Username != "" {
		clientConfig.ConnectUsername = c.config.Username
		clientConfig.ConnectPassword = []byte(c.config.Password)
	}

	// 创建连接管理器
	cm, err := mqtt.NewConnection(ctx, clientConfig)
	if err != nil {
		return fmt.Errorf("创建 MQTT 连接失败: %w", err)
	}

	c.connection = cm

	// 等待连接建立
	if err := cm.AwaitConnection(ctx); err != nil {
		return fmt.Errorf("建立 MQTT 连接失败: %w", err)
	}

	log.Printf("[MQTT] 客户端 %s 已连接", c.config.ClientID)
	return nil
}

// Subscribe 订阅 Topic
func (c *Client) Subscribe(ctx context.Context, topic string, handler MessageHandler) error {
	c.mu.Lock()
	c.handlers[topic] = handler
	c.mu.Unlock()

	_, err := c.connection.Subscribe(ctx, &mqttv5.Subscribe{
		Subscriptions: []mqttv5.SubscribeOptions{
			{Topic: topic, QoS: 1},
		},
	})

	if err != nil {
		return fmt.Errorf("订阅 %s 失败: %w", topic, err)
	}

	log.Printf("[MQTT] 已订阅: %s", topic)
	return nil
}

// SubscribeShared 订阅共享 Topic（负载均衡）
//
// 共享订阅允许多个 Hub 实例共享消息负载，适用于集群部署
func (c *Client) SubscribeShared(ctx context.Context, group, topic string, handler MessageHandler) error {
	sharedTopic := fmt.Sprintf("$share/%s/%s", group, topic)
	return c.Subscribe(ctx, sharedTopic, handler)
}

// Publish 发布消息
func (c *Client) Publish(ctx context.Context, topic string, payload []byte) error {
	_, err := c.connection.Publish(ctx, &mqttv5.Publish{
		Topic:   topic,
		Payload: payload,
		QoS:     1,
	})

	if err != nil {
		return fmt.Errorf("发布到 %s 失败: %w", topic, err)
	}

	log.Printf("[MQTT] 已发布到 %s (%d bytes)", topic, len(payload))
	return nil
}

// PublishRetain 发布保留消息
//
// 保留消息会被 Broker 保存，新订阅者会立即收到最后一条保留消息
// 适用于设备在线状态等场景
func (c *Client) PublishRetain(ctx context.Context, topic string, payload []byte) error {
	_, err := c.connection.Publish(ctx, &mqttv5.Publish{
		Topic:   topic,
		Payload: payload,
		QoS:     1,
		Retain:  true,
	})

	if err != nil {
		return fmt.Errorf("发布保留消息到 %s 失败: %w", topic, err)
	}

	log.Printf("[MQTT] 已发布保留消息到 %s", topic)
	return nil
}

// Disconnect 断开连接
func (c *Client) Disconnect(ctx context.Context) error {
	if c.connection != nil {
		log.Printf("[MQTT] 正在断开连接...")
		return c.connection.Disconnect(ctx)
	}
	return nil
}

// IsConnected 检查是否已连接
func (c *Client) IsConnected() bool {
	return c.connection != nil
}

// handleMessage 处理接收到的消息
func (c *Client) handleMessage(topic string, payload []byte) {
	c.mu.RLock()
	handler, ok := c.handlers[topic]
	c.mu.RUnlock()

	if ok {
		if err := handler(topic, payload); err != nil {
			log.Printf("[MQTT] 处理消息失败 %s: %v", topic, err)
		}
	} else {
		log.Printf("[MQTT] 未找到处理器: %s", topic)
	}
}

// mustParseURL 解析 URL，失败时 panic
func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(fmt.Sprintf("无效的 URL: %s", s))
	}
	return u
}