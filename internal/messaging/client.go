// internal/messaging/client.go - Direct Action Only
package messaging

import (
	"fmt"
	"mqtt-bridge/internal/config"
	"mqtt-bridge/internal/utils"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTClient MQTT 클라이언트 구현체
type MQTTClient struct {
	client mqtt.Client
	config *config.Config
}

// NewMQTTClient 새 MQTT 클라이언트 생성
func NewMQTTClient(cfg *config.Config) (*MQTTClient, error) {
	utils.Logger.Infof("🏗️ Creating MQTT Client")

	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.MQTTBroker)
	opts.SetClientID(cfg.MQTTClientID)
	opts.SetUsername(cfg.MQTTUsername)
	opts.SetPassword(cfg.MQTTPassword)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(10 * time.Second)

	// 연결 상태 콜백
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		utils.Logger.Info("MQTT client connected")
	})

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		utils.Logger.Errorf("MQTT connection lost: %v", err)
	})

	client := mqtt.NewClient(opts)

	// 연결 시도
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %v", token.Error())
	}

	mqttClient := &MQTTClient{
		client: client,
		config: cfg,
	}

	utils.Logger.Infof("✅ MQTT Client Created")
	return mqttClient, nil
}

// Publish 메시지 발행
func (c *MQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	if !c.client.IsConnected() {
		return fmt.Errorf("MQTT client is not connected")
	}

	// 📤 발신 메시지 상세 로깅
	var payloadStr string
	switch v := payload.(type) {
	case string:
		payloadStr = v
	case []byte:
		payloadStr = string(v)
	default:
		payloadStr = fmt.Sprintf("%v", v)
	}

	utils.Logger.Infof("📤 MQTT PUBLISH")
	utils.Logger.Infof("📤 Topic   : %s", topic)
	utils.Logger.Infof("📤 QoS    : %d, Retained: %v", qos, retained)
	utils.Logger.Infof("📤 Payload : %s", payloadStr)

	token := c.client.Publish(topic, qos, retained, payload)
	if token.Wait() && token.Error() != nil {
		utils.Logger.Errorf("❌ MQTT PUBLISH FAILED: %s - %v", topic, token.Error())
		return fmt.Errorf("failed to publish message: %v", token.Error())
	}

	utils.Logger.Infof("✅ MQTT PUBLISH SUCCESS: %s", topic)
	return nil
}

// Subscribe 토픽 구독
func (c *MQTTClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) error {
	if !c.client.IsConnected() {
		return fmt.Errorf("MQTT client is not connected")
	}

	token := c.client.Subscribe(topic, qos, callback)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to topic %s: %v", topic, token.Error())
	}

	utils.Logger.Infof("✅ Subscribed to topic: %s", topic)
	return nil
}

// Disconnect 연결 해제
func (c *MQTTClient) Disconnect(quiesce uint) {
	if c.client.IsConnected() {
		c.client.Disconnect(quiesce)
		utils.Logger.Info("MQTT client disconnected")
	}
}

// IsConnected 연결 상태 확인
func (c *MQTTClient) IsConnected() bool {
	return c.client.IsConnected()
}

// GetNativeClient 원시 클라이언트 반환
func (c *MQTTClient) GetNativeClient() mqtt.Client {
	return c.client
}

// GetConfig 설정 반환
func (c *MQTTClient) GetConfig() *config.Config {
	return c.config
}
