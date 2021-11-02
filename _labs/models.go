package _labs

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"time"
)

type Reading struct {
	Id        string    `json:"id"`
	UserId    string    `json:"userId"`
	RequestId string    `json:"requestId"`
	Child     Child     `json:"child"`
	Info      *Info     `json:"info"`
	Error     *Error    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

type Info struct {
	Message string
	Data    interface{}
}

type Error struct {
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
	StackTrace interface{} `json:"stackTrace"`
}

type Child struct {
	FnName  string    `json:"fnName"`
	Payload string    `json:"payload"`
	Type    ChildType `json:"type"`
	Caller  string    `json:"caller"`
}

type ChildType string

const (
	RemoteProcedureCall  ChildType = "rpc"
	ExternalEndpointCall ChildType = "eec"
	InternalMethodCall   ChildType = "imc"
)

func (c ChildType) String() string {
	return string(c)
}

func (c ChildType) IsValid() bool {
	switch c {
	case RemoteProcedureCall,
		ExternalEndpointCall,
		InternalMethodCall:
		return true
	default:
		return false
	}
}

func (t Reading) Validate() error {
	return validation.ValidateStruct(&t,
		validation.Field(&t.UserId, validation.Required),
		validation.Field(&t.RequestId, validation.Required),
		validation.Field(&t.Child.Type, validation.Required),
		validation.Field(&t.Child.Caller, validation.Required),
		validation.Field(&t.Child.FnName, validation.Required),
		validation.Field(&t.Child.Payload, validation.NilOrNotEmpty, validation.Required),
		validation.Field(&t.Info, validation.NilOrNotEmpty),
		validation.Field(&t.Info.Data, validation.Required, validation.NilOrNotEmpty),
	)
}

type Service struct {
	ProductLine     string
	ServiceName     string
	Instances       int
	Uptime          int64
	Heartbeat       int
	Telemetry       []*Reading
	BaseURL         string
	ServiceCategory ServiceCategory
	ServiceType     ServiceType
}

type ServiceType string

const (
	Microservice ServiceType = "microservice"

	Gateway ServiceType = "gateways"
)

type ServiceCategory string

const (
	GrpcRPC    ServiceCategory = "grpc"
	EventRPC   ServiceCategory = "EventRPC"
	GraphRPC   ServiceCategory = "GraphRPC"
	RestRPC    ServiceCategory = "RestRPC"
	APIGateway ServiceCategory = "APIGateway"
)
