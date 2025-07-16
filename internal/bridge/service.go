// internal/bridge/service.go - Direct Action Only
package bridge

import (
	"context"
	"mqtt-bridge/internal/config"
	"mqtt-bridge/internal/messaging"
	"mqtt-bridge/internal/utils"
)

// Service 간소화된 브릿지 서비스 (Direct Action 전용)
type Service struct {
	config     *config.Config
	mqttClient *messaging.MQTTClient
	subscriber *messaging.Subscriber
	handler    *messaging.DirectActionHandler
}

// NewService 새 브릿지 서비스 생성
func NewService(cfg *config.Config) (*Service, error) {
	utils.Logger.Infof("🏗️ Creating Direct Action Bridge Service")

	// MQTT 클라이언트 생성
	mqttClient, err := messaging.NewMQTTClient(cfg)
	if err != nil {
		return nil, err
	}

	// Direct Action 핸들러 생성
	handler := messaging.NewDirectActionHandler(mqttClient, cfg)

	// 구독자 생성
	subscriber := messaging.NewSubscriber(mqttClient, handler)

	service := &Service{
		config:     cfg,
		mqttClient: mqttClient,
		subscriber: subscriber,
		handler:    handler,
	}

	utils.Logger.Infof("✅ Direct Action Bridge Service Created")
	return service, nil
}

// Start 브릿지 서비스 시작
func (s *Service) Start(ctx context.Context) error {
	utils.Logger.Infof("🚀 Starting Direct Action Bridge Service")

	if err := s.subscriber.SubscribeAll(); err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		utils.Logger.Info("Context cancelled, stopping bridge service")
	}()

	return nil
}

// Stop 브릿지 서비스 중지
func (s *Service) Stop() {
	utils.Logger.Info("🛑 Stopping Direct Action Bridge Service")
	s.mqttClient.Disconnect(250)
	utils.Logger.Info("✅ Direct Action Bridge Service Stopped")
}
