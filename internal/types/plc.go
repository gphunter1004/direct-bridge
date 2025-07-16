// internal/types/plc_response.go
package types

import (
	"fmt"
	"strings"
)

// PLCResponse PLC 응답 구조체
type PLCResponse struct {
	Command string `json:"command"`
	Status  string `json:"status"`
}

// PLCResponseStatus PLC 응답 상태 열거형
const (
	PLCStatusWaiting      = "W" // Action is waiting
	PLCStatusInitializing = "I" // Action is initializing
	PLCStatusRunning      = "R" // Action is running
	PLCStatusSuccess      = "S" // Action completed successfully
	PLCStatusFailed       = "F" // Action failed
)

// NewPLCResponse 새 PLC 응답 생성
func NewPLCResponse(command, status, message string) *PLCResponse {
	return &PLCResponse{
		Command: extractBaseCommand(command),
		Status:  status,
	}
}

// ToResponseString PLC 응답을 문자열로 변환 (기존 형식: "COMMAND:STATUS")
func (r *PLCResponse) ToResponseString() string {
	return fmt.Sprintf("%s:%s", r.Command, r.Status)
}

// extractBaseCommand 기본 명령 추출 (내부 함수)
func extractBaseCommand(command string) string {
	parts := strings.Split(command, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return command
}
