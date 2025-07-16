// internal/types/order.go
package types

import (
	"time"
)

// OrderMessage AGV 오더 메시지 구조체
type OrderMessage struct {
	HeaderID      int64     `json:"headerId"`
	Timestamp     time.Time `json:"timestamp"`
	Version       string    `json:"version"`
	Manufacturer  string    `json:"manufacturer"`
	SerialNumber  string    `json:"serialNumber"`
	OrderID       string    `json:"orderId"`
	OrderUpdateID int       `json:"orderUpdateId"`
	ZoneSetID     *string   `json:"zoneSetId,omitempty"`
	Nodes         []Node    `json:"nodes"`
	Edges         []Edge    `json:"edges"`
}

// Node 노드 구조체
type Node struct {
	NodeID          string        `json:"nodeId"`
	SequenceID      int           `json:"sequenceId"`
	NodeDescription *string       `json:"nodeDescription,omitempty"`
	Released        bool          `json:"released"`
	NodePosition    *NodePosition `json:"nodePosition,omitempty"`
	Actions         []Action      `json:"actions"`
}

// NodePosition 노드 위치 구조체
type NodePosition struct {
	X                     float64  `json:"x"`
	Y                     float64  `json:"y"`
	Theta                 *float64 `json:"theta,omitempty"`
	AllowedDeviationXY    *float64 `json:"allowedDeviationXY,omitempty"`
	AllowedDeviationTheta *float64 `json:"allowedDeviationTheta,omitempty"`
	MapID                 string   `json:"mapId"`
	MapDescription        *string  `json:"mapDescription,omitempty"`
}

// Edge 엣지 구조체
type Edge struct {
	EdgeID           string      `json:"edgeId"`
	SequenceID       int         `json:"sequenceId"`
	EdgeDescription  *string     `json:"edgeDescription,omitempty"`
	Released         bool        `json:"released"`
	StartNodeID      string      `json:"startNodeId"`
	EndNodeID        string      `json:"endNodeId"`
	MaxSpeed         *float64    `json:"maxSpeed,omitempty"`
	MaxHeight        *float64    `json:"maxHeight,omitempty"`
	MinHeight        *float64    `json:"minHeight,omitempty"`
	Orientation      *float64    `json:"orientation,omitempty"`
	OrientationType  *string     `json:"orientationType,omitempty"`
	Direction        *string     `json:"direction,omitempty"`
	RotationAllowed  *bool       `json:"rotationAllowed,omitempty"`
	MaxRotationSpeed *float64    `json:"maxRotationSpeed,omitempty"`
	Length           *float64    `json:"length,omitempty"`
	Trajectory       *Trajectory `json:"trajectory,omitempty"`
	Corridor         *Corridor   `json:"corridor,omitempty"`
	Actions          []Action    `json:"actions"`
}

// Trajectory 궤적 구조체 (NURBS)
type Trajectory struct {
	Degree        int            `json:"degree"`
	KnotVector    []float64      `json:"knotVector"`
	ControlPoints []ControlPoint `json:"controlPoints"`
}

// ControlPoint 제어점 구조체
type ControlPoint struct {
	X      float64  `json:"x"`
	Y      float64  `json:"y"`
	Weight *float64 `json:"weight,omitempty"`
}

// Corridor 복도 구조체
type Corridor struct {
	LeftWidth        float64 `json:"leftWidth"`
	RightWidth       float64 `json:"rightWidth"`
	CorridorRefPoint *string `json:"corridorRefPoint,omitempty"`
}

// Action 액션 구조체
type Action struct {
	ActionType        string            `json:"actionType"`
	ActionID          string            `json:"actionId"`
	ActionDescription *string           `json:"actionDescription,omitempty"`
	BlockingType      string            `json:"blockingType"`
	ActionParameters  []ActionParameter `json:"actionParameters,omitempty"`
}

// ActionParameter 액션 파라미터 구조체
type ActionParameter struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// BlockingType 열거형
const (
	BlockingTypeNone = "NONE"
	BlockingTypeSoft = "SOFT"
	BlockingTypeHard = "HARD"
)

// OrientationType 열거형
const (
	OrientationTypeGlobal     = "GLOBAL"
	OrientationTypeTangential = "TANGENTIAL"
)

// CorridorRefPoint 열거형
const (
	CorridorRefPointKinematicCenter = "KINEMATICCENTER"
	CorridorRefPointContour         = "CONTOUR"
)

// NewOrderMessage 새 오더 메시지 생성
func NewOrderMessage(
	headerID int64,
	manufacturer, serialNumber, orderID string,
	orderUpdateID int,
) *OrderMessage {
	return &OrderMessage{
		HeaderID:      headerID,
		Timestamp:     time.Now(),
		Version:       "2.0.0",
		Manufacturer:  manufacturer,
		SerialNumber:  serialNumber,
		OrderID:       orderID,
		OrderUpdateID: orderUpdateID,
		Nodes:         make([]Node, 0),
		Edges:         make([]Edge, 0),
	}
}

// AddNode 노드 추가
func (o *OrderMessage) AddNode(node Node) {
	o.Nodes = append(o.Nodes, node)
}

// AddEdge 엣지 추가
func (o *OrderMessage) AddEdge(edge Edge) {
	o.Edges = append(o.Edges, edge)
}

// NewNode 새 노드 생성
func NewNode(nodeID string, sequenceID int, released bool) Node {
	return Node{
		NodeID:     nodeID,
		SequenceID: sequenceID,
		Released:   released,
		Actions:    make([]Action, 0),
	}
}

// AddAction 노드에 액션 추가
func (n *Node) AddAction(action Action) {
	n.Actions = append(n.Actions, action)
}

// NewAction 새 액션 생성
func NewAction(actionType, actionID, blockingType string) Action {
	return Action{
		ActionType:       actionType,
		ActionID:         actionID,
		BlockingType:     blockingType,
		ActionParameters: make([]ActionParameter, 0),
	}
}

// AddParameter 액션에 파라미터 추가
func (a *Action) AddParameter(key string, value interface{}) {
	a.ActionParameters = append(a.ActionParameters, ActionParameter{
		Key:   key,
		Value: value,
	})
}

// NewEdge 새 엣지 생성
func NewEdge(edgeID string, sequenceID int, released bool, startNodeID, endNodeID string) Edge {
	return Edge{
		EdgeID:      edgeID,
		SequenceID:  sequenceID,
		Released:    released,
		StartNodeID: startNodeID,
		EndNodeID:   endNodeID,
		Actions:     make([]Action, 0),
	}
}

// AddAction 엣지에 액션 추가
func (e *Edge) AddAction(action Action) {
	e.Actions = append(e.Actions, action)
}
