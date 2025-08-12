// internal/messaging/subscriber.go - Direct Action Only + Connection Handler
package messaging

import (
	"fmt"
	"mqtt-bridge/internal/utils"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Subscriber MQTT 구독 관리자
type Subscriber struct {
	client  *MQTTClient
	handler *DirectActionHandler
}

// NewSubscriber 새 구독자 생성
func NewSubscriber(client *MQTTClient, handler *DirectActionHandler) *Subscriber {
	utils.Logger.Infof("🏗️ Creating MQTT Subscriber")

	subscriber := &Subscriber{
		client:  client,
		handler: handler,
	}

	utils.Logger.Infof("✅ MQTT Subscriber Created")
	return subscriber
}

// SubscribeAll 필요한 토픽들 구독
func (s *Subscriber) SubscribeAll() error {
	utils.Logger.Infof("🔔 Starting Subscriptions")

	// 구독할 토픽들
	subscriptions := []struct {
		topic       string
		description string
		handler     mqtt.MessageHandler
	}{
		{
			topic:       "bridge/command",
			description: "PLC Commands",
			handler:     s.handlePLCCommand,
		},
		{
			topic:       "meili/v2/+/+/state",
			description: "Robot States",
			handler:     s.handleRobotState,
		},
		{
			topic:       "meili/v2/+/+/connection",
			description: "Robot Connection States",
			handler:     s.handleRobotConnection,
		},
	}

	// 각 토픽 구독
	for _, sub := range subscriptions {
		utils.Logger.Infof("🔔 Subscribing to: %s (%s)", sub.topic, sub.description)

		err := s.client.Subscribe(sub.topic, 0, sub.handler)
		if err != nil {
			utils.Logger.Errorf("❌ Subscription failed: %s - %v", sub.topic, err)
			return fmt.Errorf("failed to subscribe to %s: %v", sub.topic, err)
		}

		utils.Logger.Infof("✅ Subscription success: %s", sub.topic)
	}

	utils.Logger.Infof("🎉 All subscriptions completed")
	return nil
}

// handlePLCCommand PLC 명령 메시지 처리
func (s *Subscriber) handlePLCCommand(client mqtt.Client, msg mqtt.Message) {
	utils.Logger.Infof("📨 MQTT RECEIVED")
	utils.Logger.Infof("📨 Topic   : %s", msg.Topic())
	utils.Logger.Infof("📨 QoS    : %d, MessageID: %d", msg.Qos(), msg.MessageID())
	utils.Logger.Infof("📨 Payload : %s", string(msg.Payload()))

	s.handler.HandlePLCCommand(client, msg)
}

// handleRobotState 로봇 상태 메시지 처리
func (s *Subscriber) handleRobotState(client mqtt.Client, msg mqtt.Message) {
	// 로봇 상태 메시지도 전체 페이로드 출력 (줄이지 않음)
	utils.Logger.Infof("📨 MQTT RECEIVED")
	utils.Logger.Infof("📨 Topic   : %s", msg.Topic())
	utils.Logger.Infof("📨 QoS    : %d, MessageID: %d", msg.Qos(), msg.MessageID())
	utils.Logger.Infof("📨 Payload : %s", string(msg.Payload()))

	s.handler.HandleRobotState(client, msg)
}

// handleRobotConnection 로봇 연결 상태 메시지 처리
func (s *Subscriber) handleRobotConnection(client mqtt.Client, msg mqtt.Message) {
	utils.Logger.Infof("📨 MQTT RECEIVED")
	utils.Logger.Infof("📨 Topic   : %s", msg.Topic())
	utils.Logger.Infof("📨 QoS    : %d, MessageID: %d", msg.Qos(), msg.MessageID())
	utils.Logger.Infof("📨 Payload : %s", string(msg.Payload()))

	s.handler.HandleRobotConnection(client, msg)
}
