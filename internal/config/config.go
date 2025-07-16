// internal/config/config.go - Direct Action Only
package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// MQTT
	MQTTBroker       string
	MQTTPort         string
	MQTTClientID     string
	MQTTUsername     string
	MQTTPassword     string
	PlcResponseTopic string

	// Robot Configuration
	RobotSerialNumber string
	RobotManufacturer string

	// Application
	LogLevel string
	Timeout  time.Duration
}

func Load() (*Config, error) {
	// .env 파일 로드 (선택적)
	if err := godotenv.Load(); err != nil {
		// .env 파일이 없어도 계속 진행
	}

	return &Config{
		MQTTBroker:        getEnv("MQTT_BROKER", "tcp://localhost:1883"),
		MQTTPort:          getEnv("MQTT_PORT", "1883"),
		MQTTClientID:      getEnv("MQTT_CLIENT_ID", "DEX0002_DIRECT_BRIDGE"),
		MQTTUsername:      getEnv("MQTT_USERNAME", "DEX0002_DIRECT_BRIDGE"),
		MQTTPassword:      getEnv("MQTT_PASSWORD", "DEX0002_DIRECT_BRIDGE"),
		PlcResponseTopic:  getEnv("PLC_RESPONSE_TOPIC", "bridge/response"),
		RobotSerialNumber: getEnv("ROBOT_SERIAL_NUMBER", "DEX0002"),
		RobotManufacturer: getEnv("ROBOT_MANUFACTURER", "Roboligent"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		Timeout:           30 * time.Second,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
