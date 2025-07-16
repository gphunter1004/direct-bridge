// internal/types/instant_actions.go
package types

import (
	"time"
)

// InstantActionsMessage InstantActions 메시지 구조체
type InstantActionsMessage struct {
	HeaderID     int64           `json:"headerId"`
	Timestamp    time.Time       `json:"timestamp"`
	Version      string          `json:"version"`
	Manufacturer string          `json:"manufacturer"`
	SerialNumber string          `json:"serialNumber"`
	Actions      []InstantAction `json:"actions"`
}

// InstantAction InstantAction 구조체
type InstantAction struct {
	ActionType        string                   `json:"actionType"`
	ActionID          string                   `json:"actionId"`
	ActionDescription *string                  `json:"actionDescription,omitempty"`
	BlockingType      string                   `json:"blockingType"`
	ActionParameters  []InstantActionParameter `json:"actionParameters,omitempty"`
}

// InstantActionParameter InstantAction 파라미터 구조체
type InstantActionParameter struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// NewInstantActionsMessage 새 InstantActions 메시지 생성
func NewInstantActionsMessage(
	headerID int64,
	manufacturer, serialNumber string,
) *InstantActionsMessage {
	return &InstantActionsMessage{
		HeaderID:     headerID,
		Timestamp:    time.Now(),
		Version:      "2.0.0",
		Manufacturer: manufacturer,
		SerialNumber: serialNumber,
		Actions:      make([]InstantAction, 0),
	}
}

// AddAction InstantActions 메시지에 액션 추가
func (i *InstantActionsMessage) AddAction(action InstantAction) {
	i.Actions = append(i.Actions, action)
}

// NewInstantAction 새 InstantAction 생성
func NewInstantAction(actionType, actionID, blockingType string) InstantAction {
	return InstantAction{
		ActionType:       actionType,
		ActionID:         actionID,
		BlockingType:     blockingType,
		ActionParameters: make([]InstantActionParameter, 0),
	}
}

// AddParameter InstantAction에 파라미터 추가
func (a *InstantAction) AddParameter(key string, value interface{}) {
	a.ActionParameters = append(a.ActionParameters, InstantActionParameter{
		Key:   key,
		Value: value,
	})
}
