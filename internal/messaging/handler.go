// internal/messaging/handler.go - Direct Action Only (구조체 사용)
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
	mqttClient     *MQTTClient
	config         *config.Config
	activeOrders   map[string]string // orderID -> original command mapping
	canceledOrders map[string]string // orderID -> original cancel command mapping (취소된 오더 추적)
}

// NewDirectActionHandler 새 Direct Action 핸들러 생성
func NewDirectActionHandler(mqttClient *MQTTClient, cfg *config.Config) *DirectActionHandler {
	utils.Logger.Infof("🏗️ Creating Direct Action Handler")

	handler := &DirectActionHandler{
		mqttClient:     mqttClient,
		config:         cfg,
		activeOrders:   make(map[string]string),
		canceledOrders: make(map[string]string),
	}

	utils.Logger.Infof("✅ Direct Action Handler Created")
	return handler
}

// HandlePLCCommand PLC 명령 처리 (Direct Action만)
func (h *DirectActionHandler) HandlePLCCommand(client mqtt.Client, msg mqtt.Message) {
	commandStr := strings.TrimSpace(string(msg.Payload()))
	utils.Logger.Infof("🎯 PLC Command received: '%s'", commandStr)

	// 취소 명령 확인
	if h.isCancelCommand(commandStr) {
		h.handleCancelCommand(commandStr)
		return
	}

	// Direct Action 명령인지 확인
	if !h.isDirectActionCommand(commandStr) {
		utils.Logger.Errorf("❌ Non-direct action command rejected: %s", commandStr)
		h.sendPLCResponse(commandStr, types.PLCStatusFailed)
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

	// OrderID 확인
	orderID, hasOrderID := stateMsg["orderId"].(string)
	if hasOrderID && orderID != "" {
		actionStates, hasActions := stateMsg["actionStates"].([]interface{})

		// 취소된 오더인지 확인 (PLC 취소 요청한 경우)
		if originalCancelCommand, exists := h.canceledOrders[orderID]; exists {
			if hasActions {
				utils.Logger.Infof("🔍 Processing canceled order states for OrderID: %s", orderID)
				h.processCanceledOrderStates(orderID, originalCancelCommand, actionStates)
			}
			return
		}

		// 활성 오더 처리 (일반 실행 중이거나 로봇 자체 취소된 경우)
		originalCommand, exists := h.activeOrders[orderID]
		if exists {
			if hasActions {
				utils.Logger.Infof("🔍 Processing action states for OrderID: %s (Command: %s)", orderID, originalCommand)
				h.processActionStates(orderID, originalCommand, actionStates)
			}
		}
	}
}

// isDirectActionCommand Direct Action 명령인지 확인
func (h *DirectActionHandler) isDirectActionCommand(commandStr string) bool {
	return strings.HasSuffix(commandStr, ":I") || strings.Contains(commandStr, ":T")
}

// isCancelCommand 취소 명령인지 확인
func (h *DirectActionHandler) isCancelCommand(commandStr string) bool {
	return strings.HasSuffix(commandStr, ":C")
}

// handleDirectAction Direct Action 처리
func (h *DirectActionHandler) handleDirectAction(commandStr string) {
	parts := strings.Split(commandStr, ":")
	if len(parts) < 2 {
		h.sendPLCResponse(commandStr, types.PLCStatusFailed)
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
		h.sendPLCResponse(commandStr, types.PLCStatusFailed)
		return
	}

	// OrderID와 원본 명령 매핑 저장
	h.activeOrders[orderID] = commandStr

	utils.Logger.Infof("✅ Direct action order sent: %s (OrderID: %s)", commandStr, orderID)
}

// handleCancelCommand 취소 명령 처리
func (h *DirectActionHandler) handleCancelCommand(commandStr string) {
	baseCommand := h.extractBaseCommand(commandStr)

	// 해당 명령에 대한 활성 오더 찾기
	var targetOrderID string
	for orderID, originalCommand := range h.activeOrders {
		if h.extractBaseCommand(originalCommand) == baseCommand {
			targetOrderID = orderID
			break
		}
	}

	if targetOrderID == "" {
		utils.Logger.Warnf("⚠️ No active order found for command: %s", baseCommand)
		h.sendPLCResponse(commandStr, types.PLCStatusFailed)
		return
	}

	// InstantActions로 취소 명령 전송
	if err := h.sendCancelOrder(targetOrderID); err != nil {
		utils.Logger.Errorf("❌ Failed to send cancel order: %v", err)
		h.sendPLCResponse(commandStr, types.PLCStatusFailed)
		return
	}

	// 활성 오더에서 제거하고 취소된 오더로 이동
	delete(h.activeOrders, targetOrderID)
	h.canceledOrders[targetOrderID] = commandStr

	utils.Logger.Infof("✅ Cancel order sent for: %s (OrderID: %s)", baseCommand, targetOrderID)
}

// sendDirectActionOrder Direct Action 오더 전송 (구조체 사용)
func (h *DirectActionHandler) sendDirectActionOrder(baseCommand string, commandType rune, armParam string) (string, error) {
	// 액션 타입과 파라미터 결정
	actionType, actionParameters := h.buildActionParameters(baseCommand, commandType, armParam)
	if actionType == "" {
		return "", fmt.Errorf("invalid direct action command type: %c", commandType)
	}

	// ID 생성
	orderID := h.generateOrderID()
	nodeID := h.generateNodeID()
	actionID := h.generateActionID()

	// 오더 생성
	order := h.buildOrder(orderID, nodeID, actionID, baseCommand, actionType, actionParameters)

	// JSON 마샬링 및 전송
	return h.publishOrder(order, orderID, actionType, baseCommand)
}

// buildActionParameters 액션 파라미터 구성
func (h *DirectActionHandler) buildActionParameters(baseCommand string, commandType rune, armParam string) (string, []types.ActionParameter) {
	switch commandType {
	case 'I':
		return "Roboligent Robin - Inference", []types.ActionParameter{
			{Key: "inference_name", Value: baseCommand},
		}
	case 'T':
		return "Roboligent Robin - Follow Trajectory", []types.ActionParameter{
			{Key: "trajectory_name", Value: baseCommand},
			{Key: "arm", Value: h.parseArmParam(armParam)},
		}
	default:
		return "", nil
	}
}

// buildOrder 오더 구조체 생성
func (h *DirectActionHandler) buildOrder(orderID, nodeID, actionID, baseCommand, actionType string, actionParameters []types.ActionParameter) *types.OrderMessage {
	// 오더 생성
	order := types.NewOrderMessage(
		h.getNextHeaderID(),
		h.config.RobotManufacturer,
		h.config.RobotSerialNumber,
		orderID,
		0,
	)

	// 노드 생성 및 설정
	node := types.NewNode(nodeID, 1, true)
	nodeDescription := fmt.Sprintf("Direct action for command %s", baseCommand)
	node.NodeDescription = &nodeDescription
	node.NodePosition = h.createDefaultNodePosition()

	// 액션 생성 및 설정
	action := types.NewAction(actionType, actionID, types.BlockingTypeNone)
	actionDescription := fmt.Sprintf("Execute %s for %s", actionType, baseCommand)
	action.ActionDescription = &actionDescription
	action.ActionParameters = actionParameters

	// 노드에 액션 추가, 오더에 노드 추가
	node.AddAction(action)
	order.AddNode(node)

	return order
}

// createDefaultNodePosition 기본 노드 위치 생성
func (h *DirectActionHandler) createDefaultNodePosition() *types.NodePosition {
	theta := 0.0
	allowedDeviationXY := 0.0
	allowedDeviationTheta := 0.0
	mapDescription := ""

	return &types.NodePosition{
		X:                     0.0,
		Y:                     0.0,
		Theta:                 &theta,
		AllowedDeviationXY:    &allowedDeviationXY,
		AllowedDeviationTheta: &allowedDeviationTheta,
		MapID:                 "",
		MapDescription:        &mapDescription,
	}
}

// publishOrder 오더 발행
func (h *DirectActionHandler) publishOrder(order *types.OrderMessage, orderID, actionType, baseCommand string) (string, error) {
	msgData, err := json.Marshal(order)
	if err != nil {
		return "", fmt.Errorf("failed to marshal order: %v", err)
	}

	topic := fmt.Sprintf("meili/v2/%s/%s/order", h.config.RobotManufacturer, h.config.RobotSerialNumber)

	utils.Logger.Infof("📤 Sending Robot Order to: %s", topic)
	utils.Logger.Infof("📤 Order Details: OrderID=%s, ActionType=%s, BaseCommand=%s", orderID, actionType, baseCommand)

	if err := h.mqttClient.Publish(topic, 0, false, msgData); err != nil {
		return "", err
	}

	utils.Logger.Infof("✅ Robot Order sent successfully: OrderID=%s", orderID)
	return orderID, nil
}

// sendCancelOrder InstantActions로 취소 명령 전송
func (h *DirectActionHandler) sendCancelOrder(orderID string) error {
	// InstantActions 메시지 생성
	instantActions := types.NewInstantActionsMessage(
		h.getNextHeaderID(),
		h.config.RobotManufacturer,
		h.config.RobotSerialNumber,
	)

	// 취소 액션 생성
	actionID := h.generateActionID()
	cancelAction := types.NewInstantAction("cancelOrder", actionID, types.BlockingTypeHard)

	// InstantActions에 액션 추가
	instantActions.AddAction(cancelAction)

	// JSON 마샬링
	msgData, err := json.Marshal(instantActions)
	if err != nil {
		return fmt.Errorf("failed to marshal instant actions: %v", err)
	}

	// 전송
	topic := fmt.Sprintf("meili/v2/%s/%s/instantActions", h.config.RobotManufacturer, h.config.RobotSerialNumber)

	utils.Logger.Infof("📤 Sending Cancel Order via InstantActions to: %s", topic)
	utils.Logger.Infof("📤 Cancel Details: OrderID=%s, ActionID=%s", orderID, actionID)

	if err := h.mqttClient.Publish(topic, 0, false, msgData); err != nil {
		return err
	}

	utils.Logger.Infof("✅ Cancel order sent successfully via InstantActions")
	return nil
}

// processActionStates 액션 상태 처리
func (h *DirectActionHandler) processActionStates(orderID, originalCommand string, actionStates []interface{}) {
	// 액션 상태들을 확인하여 전체 상태 결정
	statusCounts := make(map[string]int)

	for _, actionState := range actionStates {
		if actionMap, ok := actionState.(map[string]interface{}); ok {
			if actionStatus, hasStatus := actionMap["actionStatus"].(string); hasStatus {
				statusCounts[actionStatus]++
				if actionID, _ := actionMap["actionId"].(string); actionID != "" {
					utils.Logger.Infof("🔍 Action %s status: %s", actionID, actionStatus)
				}
			}
		}
	}

	// 상태에 따른 응답 결정 및 전송 (우선순위 순서)
	switch {
	case statusCounts["FAILED"] > 0:
		utils.Logger.Errorf("❌ Action failed for OrderID: %s", orderID)
		h.sendPLCResponse(originalCommand, types.PLCStatusFailed)
		delete(h.activeOrders, orderID)
	case statusCounts["FINISHED"] > 0 && statusCounts["RUNNING"] == 0 && statusCounts["INITIALIZING"] == 0 && statusCounts["WAITING"] == 0:
		utils.Logger.Infof("✅ All actions finished for OrderID: %s", orderID)
		h.sendPLCResponse(originalCommand, types.PLCStatusSuccess)
		delete(h.activeOrders, orderID)
	case statusCounts["RUNNING"] > 0:
		utils.Logger.Infof("🏃 Action running for OrderID: %s", orderID)
		h.sendPLCResponse(originalCommand, types.PLCStatusRunning)
	case statusCounts["INITIALIZING"] > 0:
		utils.Logger.Infof("🔄 Action initializing for OrderID: %s", orderID)
		h.sendPLCResponse(originalCommand, types.PLCStatusInitializing)
	case statusCounts["WAITING"] > 0:
		utils.Logger.Infof("⏳ Action waiting for OrderID: %s", orderID)
		h.sendPLCResponse(originalCommand, types.PLCStatusWaiting)
	}
}

// processCanceledOrderStates 취소된 오더 상태 처리 (PLC 취소 요청 후)
func (h *DirectActionHandler) processCanceledOrderStates(orderID, originalCancelCommand string, actionStates []interface{}) {
	// 취소된 오더의 액션 상태에 따라 취소 명령에 대한 응답 처리
	for _, actionState := range actionStates {
		if actionMap, ok := actionState.(map[string]interface{}); ok {
			if actionStatus, hasStatus := actionMap["actionStatus"].(string); hasStatus {
				actionID, _ := actionMap["actionId"].(string)
				utils.Logger.Infof("🔍 Canceled Order Action %s status: %s", actionID, actionStatus)

				switch actionStatus {
				case "FAILED":
					utils.Logger.Infof("✅ Canceled order action failed as expected: %s", orderID)
					h.sendPLCResponse(originalCancelCommand, types.PLCStatusFailed)
					delete(h.canceledOrders, orderID)
					return
				case "FINISHED":
					utils.Logger.Infof("✅ Canceled order action finished: %s", orderID)
					h.sendPLCResponse(originalCancelCommand, types.PLCStatusSuccess)
					delete(h.canceledOrders, orderID)
					return
				}
			}
		}
	}
}

// sendPLCResponse PLC에 응답 전송 (구조체 사용)
func (h *DirectActionHandler) sendPLCResponse(command, status string) {
	// PLC 응답 구조체 생성
	plcResponse := types.NewPLCResponse(command, status, "")

	// 기존 형식의 응답 문자열 생성 (COMMAND:STATUS)
	responseStr := plcResponse.ToResponseString()

	utils.Logger.Infof("📤 MQTT PUBLISH")
	utils.Logger.Infof("📤 Topic   : %s", h.config.PlcResponseTopic)
	utils.Logger.Infof("📤 QoS    : %d, Retained: %v", 0, false)
	utils.Logger.Infof("📤 Payload : %s", responseStr)

	// MQTTClient.Publish에서 이미 성공/실패 로그를 모두 출력하므로 여기서는 제거
	h.mqttClient.Publish(h.config.PlcResponseTopic, 0, false, responseStr)
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
