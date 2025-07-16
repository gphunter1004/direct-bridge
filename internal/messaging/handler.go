// internal/messaging/handler.go - Direct Action Only
package messaging

import (
	"encoding/json"
	"fmt"
	"mqtt-bridge/internal/config"
	"mqtt-bridge/internal/types"
	"mqtt-bridge/internal/utils"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// DirectActionHandler Direct Action 처리 핸들러
type DirectActionHandler struct {
	mqttClient   *MQTTClient
	config       *config.Config
	activeOrders map[string]string // orderID -> original command mapping
}

// NewDirectActionHandler 새 Direct Action 핸들러 생성
func NewDirectActionHandler(mqttClient *MQTTClient, cfg *config.Config) *DirectActionHandler {
	utils.Logger.Infof("🏗️ Creating Direct Action Handler")

	handler := &DirectActionHandler{
		mqttClient:   mqttClient,
		config:       cfg,
		activeOrders: make(map[string]string),
	}

	utils.Logger.Infof("✅ Direct Action Handler Created")
	return handler
}

// HandlePLCCommand PLC 명령 처리 (Direct Action만)
func (h *DirectActionHandler) HandlePLCCommand(client mqtt.Client, msg mqtt.Message) {
	commandStr := strings.TrimSpace(string(msg.Payload()))
	utils.Logger.Infof("🎯 PLC Command received: '%s'", commandStr)

	// Direct Action 명령인지 확인
	if !h.isDirectActionCommand(commandStr) {
		utils.Logger.Errorf("❌ Non-direct action command rejected: %s", commandStr)
		h.sendPLCResponse(commandStr, "F", "Only direct action commands are supported")
		return
	}

	// Direct Action 처리
	h.handleDirectAction(commandStr)
}

// HandleRobotState 로봇 상태 메시지 처리
func (h *DirectActionHandler) HandleRobotState(client mqtt.Client, msg mqtt.Message) {
	utils.Logger.Debugf("📊 Processing robot state message")

	var stateMsg map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &stateMsg); err != nil {
		utils.Logger.Errorf("❌ Failed to parse robot state: %v", err)
		return
	}

	// OrderID가 있고 액션 상태가 있는지 확인
	orderID, hasOrderID := stateMsg["orderId"].(string)
	actionStates, hasActions := stateMsg["actionStates"].([]interface{})

	if hasOrderID && hasActions && orderID != "" {
		utils.Logger.Infof("📊 Robot State Analysis: OrderID=%s, ActionCount=%d", orderID, len(actionStates))
		h.processActionStates(orderID, actionStates)
	} else {
		utils.Logger.Debugf("📊 Robot state without relevant order/action data")
	}
}

// isDirectActionCommand Direct Action 명령인지 확인
func (h *DirectActionHandler) isDirectActionCommand(commandStr string) bool {
	return strings.HasSuffix(commandStr, ":I") || strings.Contains(commandStr, ":T")
}

// handleDirectAction Direct Action 처리
func (h *DirectActionHandler) handleDirectAction(commandStr string) {
	parts := strings.Split(commandStr, ":")
	if len(parts) < 2 {
		h.sendPLCResponse(commandStr, "F", "Invalid command format")
		return
	}

	baseCommand := parts[0]
	cmdType := rune(parts[1][0])
	armParam := ""
	if len(parts) >= 3 {
		armParam = parts[2]
	}

	// Direct Action 오더 전송
	orderID, err := h.sendDirectActionOrder(baseCommand, cmdType, armParam)
	if err != nil {
		utils.Logger.Errorf("❌ Failed to send direct action order: %v", err)
		h.sendPLCResponse(commandStr, "F", "Failed to send order to robot")
		return
	}

	// OrderID와 원본 명령 매핑 저장
	h.activeOrders[orderID] = commandStr

	utils.Logger.Infof("✅ Direct action order sent: %s (OrderID: %s)", commandStr, orderID)
	utils.Logger.Infof("📝 Waiting for robot state to send K response...")
}

// sendDirectActionOrder Direct Action 오더 전송
func (h *DirectActionHandler) sendDirectActionOrder(baseCommand string, commandType rune, armParam string) (string, error) {
	var actionType string
	var actionParameters []map[string]interface{}

	switch commandType {
	case 'I':
		actionType = "Roboligent Robin - Inference"
		actionParameters = []map[string]interface{}{
			{
				"key":   "inference_name",
				"value": baseCommand,
			},
		}
	case 'T':
		actionType = "Roboligent Robin - Follow Trajectory"
		actionParameters = []map[string]interface{}{
			{
				"key":   "trajectory_name",
				"value": baseCommand,
			},
		}

		// arm 파라미터 처리
		arm := h.parseArmParam(armParam)
		actionParameters = append(actionParameters, map[string]interface{}{
			"key":   "arm",
			"value": arm,
		})

	default:
		return "", fmt.Errorf("invalid direct action command type: %c", commandType)
	}

	orderID := h.generateOrderID()
	nodeID := h.generateNodeID()
	actionID := h.generateActionID()

	directOrder := map[string]interface{}{
		"headerId":      h.getNextHeaderID(),
		"timestamp":     time.Now().Format(time.RFC3339Nano),
		"version":       "2.0.0",
		"manufacturer":  h.config.RobotManufacturer,
		"serialNumber":  h.config.RobotSerialNumber,
		"orderId":       orderID,
		"orderUpdateId": 0,
		"nodes": []map[string]interface{}{
			{
				"nodeId":      nodeID,
				"description": fmt.Sprintf("Direct action for command %s", baseCommand),
				"sequenceId":  1,
				"released":    true,
				"nodePosition": map[string]interface{}{
					"x":                     types.ZeroFloat64(),
					"y":                     types.ZeroFloat64(),
					"theta":                 types.ZeroFloat64(),
					"allowedDeviationXY":    types.ZeroFloat64(),
					"allowedDeviationTheta": types.ZeroFloat64(),
					"mapId":                 "",
				},
				"actions": []map[string]interface{}{
					{
						"actionType":        actionType,
						"actionId":          actionID,
						"actionDescription": fmt.Sprintf("Execute %s for %s", actionType, baseCommand),
						"blockingType":      "NONE",
						"actionParameters":  actionParameters,
					},
				},
			},
		},
		"edges": []map[string]interface{}{},
	}

	// 오더 전송
	topic := fmt.Sprintf("meili/v2/%s/%s/order", h.config.RobotManufacturer, h.config.RobotSerialNumber)
	msgData, err := json.Marshal(directOrder)
	if err != nil {
		return "", fmt.Errorf("failed to marshal order: %v", err)
	}

	utils.Logger.Infof("📤 Sending Robot Order to: %s", topic)
	utils.Logger.Infof("📤 Order Details: OrderID=%s, ActionType=%s, BaseCommand=%s", orderID, actionType, baseCommand)

	if err := h.mqttClient.Publish(topic, 0, false, msgData); err != nil {
		return "", err
	}

	utils.Logger.Infof("✅ Robot Order sent successfully: OrderID=%s", orderID)
	return orderID, nil
}

// processActionStates 액션 상태 처리
func (h *DirectActionHandler) processActionStates(orderID string, actionStates []interface{}) {
	// 활성 오더인지 확인
	originalCommand, exists := h.activeOrders[orderID]
	if !exists {
		utils.Logger.Debugf("🔍 OrderID %s not in active orders, skipping", orderID)
		return
	}

	utils.Logger.Infof("🔍 Processing action states for OrderID: %s (Command: %s)", orderID, originalCommand)

	// 액션 상태들을 확인하여 전체 상태 결정
	hasRunning := false
	hasFinished := false
	hasFailed := false
	hasWaiting := false

	for _, actionState := range actionStates {
		if actionMap, ok := actionState.(map[string]interface{}); ok {
			actionStatus, hasStatus := actionMap["actionStatus"].(string)
			actionID, _ := actionMap["actionId"].(string)

			if hasStatus {
				utils.Logger.Infof("🔍 Action %s status: %s", actionID, actionStatus)

				switch actionStatus {
				case "WAITING", "INITIALIZING":
					hasWaiting = true
				case "RUNNING":
					hasRunning = true
				case "FINISHED":
					hasFinished = true
				case "FAILED":
					hasFailed = true
				}
			}
		}
	}

	// 상태에 따른 응답 결정 및 전송
	if hasFailed {
		utils.Logger.Errorf("❌ Action failed for OrderID: %s", orderID)
		h.sendPLCResponse(originalCommand, "F", "Action failed")
		delete(h.activeOrders, orderID) // 완료된 오더 제거
	} else if hasFinished && !hasRunning && !hasWaiting {
		utils.Logger.Infof("✅ All actions finished for OrderID: %s", orderID)
		h.sendPLCResponse(originalCommand, "S", "Action completed successfully")
		delete(h.activeOrders, orderID) // 완료된 오더 제거
	} else if hasRunning {
		utils.Logger.Infof("🏃 Action running for OrderID: %s", orderID)
		h.sendPLCResponse(originalCommand, "R", "Action is running")
		// 실행 중이므로 오더는 유지
	} else if hasWaiting {
		// 처음 WAITING 상태일 때 K(Acknowledged) 응답
		utils.Logger.Infof("⏳ Action acknowledged for OrderID: %s", orderID)
		h.sendPLCResponse(originalCommand, "K", "Order acknowledged by robot")
		// 대기 중이므로 오더는 유지
	}
}

// sendPLCResponse PLC에 응답 전송
func (h *DirectActionHandler) sendPLCResponse(command, status, message string) {
	response := fmt.Sprintf("%s:%s", h.extractBaseCommand(command), status)

	if status == "F" && message != "" {
		utils.Logger.Errorf("Command %s failed: %s", command, message)
	}

	utils.Logger.Infof("📤 Sending PLC Response: %s", response)

	if err := h.mqttClient.Publish(h.config.PlcResponseTopic, 0, false, response); err != nil {
		utils.Logger.Errorf("❌ Failed to send PLC response: %v", err)
	} else {
		utils.Logger.Infof("✅ PLC Response sent successfully: %s", response)
	}
}

// extractBaseCommand 기본 명령 추출
func (h *DirectActionHandler) extractBaseCommand(command string) string {
	parts := strings.Split(command, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return command
}

// parseArmParam 팔 파라미터 파싱
func (h *DirectActionHandler) parseArmParam(armParam string) string {
	switch armParam {
	case "R", "":
		return "right"
	case "L":
		return "left"
	default:
		return "right" // 기본값
	}
}

// ID 생성 헬퍼 함수들
func (h *DirectActionHandler) generateOrderID() string {
	return fmt.Sprintf("%016x", time.Now().UnixNano())
}

func (h *DirectActionHandler) generateNodeID() string {
	return fmt.Sprintf("%016x", time.Now().UnixNano()+1)
}

func (h *DirectActionHandler) generateActionID() string {
	return fmt.Sprintf("%016x", time.Now().UnixNano()+2)
}

var headerIDCounter int64

func (h *DirectActionHandler) getNextHeaderID() int64 {
	headerIDCounter++
	return headerIDCounter
}
