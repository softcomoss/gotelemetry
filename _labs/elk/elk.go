package elk

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/softcomoss/jetstreamclient"
	telemetry "github.com/softcomoss/telemtry"
	"go.elastic.co/apm/module/apmlogrus"
	"google.golang.org/grpc"
)

type elkTelemetryService struct {
	*logrus.Logger
	eventStore jetstreamclient.EventStore
}

func (e *elkTelemetryService) Client() interface{} {
	panic("implement me")
}

func (e *elkTelemetryService) RecordServiceReadings(ctx context.Context, readings ...*telemetry.Reading) error {
	if ctx == nil {
		ctx = context.Background()
	}

	//for _, reading := range readings {
	//	ctx.Deadline()
	//}

	traceContextFields := apmlogrus.TraceContext(ctx)
	e.WithFields(traceContextFields).Debug("handling request")
	return nil
}

func (e *elkTelemetryService) RecordOutgoingEvent(topic string, data []byte) error {
	panic("implement me")
}

func (e *elkTelemetryService) RecordIncomingEvents() error {
	panic("implement me")
}

func (e *elkTelemetryService) RecordGRPCWithClientUnaryInterceptor(f telemetry.UnaryClientInterceptor) grpc.DialOption {
	panic("implement me")
}

func (e *elkTelemetryService) RecordGRPCWithServerUnaryInterceptor(f telemetry.UnaryServerInterceptor) grpc.ServerOption {
	panic("implement me")
}

func NewElkTelemetryService(opts ...*telemetry.Options) telemetry.Telemetry {
	logrus.AddHook(&apmlogrus.Hook{})
	return &elkTelemetryService{
		logrus.New(),
	}
}
