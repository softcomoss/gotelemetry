package gotelemetry

import (
	"context"
	"errors"
	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpcctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/sirupsen/logrus"
	grpclogrus "github.com/softcomoss/gotelemetry/libs/logrus"
	"github.com/softcomoss/jetstreamclient"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmgrpc"
	"go.elastic.co/apm/module/apmmongo"
	"go.elastic.co/ecslogrus"
	"go.mongodb.org/mongo-driver/event"
	"google.golang.org/grpc"
	"log"
	"os"
)

type Options struct {
	grpcServerOptions []grpc.ServerOption
	eventStore        jetstreamclient.EventStore
	formatter         logrus.Formatter
	loggerHooks       []logrus.Hook
	logFile           *os.File
}

type Option func(o *Options) error

func SetGRPCServerInterceptors(opt ...grpc.ServerOption) Option {
	return func(o *Options) error {
		if o.grpcServerOptions == nil {
			o.grpcServerOptions = make([]grpc.ServerOption, 0)
		}

		o.grpcServerOptions = append(o.grpcServerOptions, opt...)
		return nil
	}
}

func SetServerLogFormatter(formatter logrus.Formatter) Option {
	return func(o *Options) error {
		if formatter == nil {
			return errors.New("log formatter must not be nil")
		}

		o.formatter = formatter
		return nil
	}
}

func EnableServerFileLogging(formatter logrus.Formatter) Option {
	return func(o *Options) error {
		if formatter == nil {
			return errors.New("log formatter must not be nil")
		}

		o.formatter = formatter
		return nil
	}
}

func SetServerLogHook(hook ...logrus.Hook) Option {
	return func(o *Options) error {
		if o.loggerHooks == nil {
			o.loggerHooks = make([]logrus.Hook, 0)
		}

		o.loggerHooks = append(o.loggerHooks, hook...)
		return nil
	}
}

func SetServerEventStore(eventStore jetstreamclient.EventStore) Option {
	return func(o *Options) error {
		if eventStore == nil {
			return errors.New("invalid eventStore")
		}

		o.eventStore = eventStore
		return nil
	}
}

func mergeServerOptions(options ...Option) *Options {
	opt := &Options{
		formatter:   &ecslogrus.Formatter{PrettyPrint: true},
		loggerHooks: make([]logrus.Hook, 0),
		eventStore:  nil,
	}

	for _, option := range options {
		if err := option(opt); err != nil {
			log.Fatal(err)
		}
	}

	return opt
}

type SoftcomTelemetry struct {
	serviceName string
	grpcServer  *grpc.Server
	//httpServer *http.Server
	*logrus.Logger
	jetstreamclient.EventStore
}

func (s SoftcomTelemetry) UseInterceptedGRPCServer() *grpc.Server {
	return s.grpcServer
}

func (s SoftcomTelemetry) UseInterceptedLogger() *logrus.Logger {
	return s.Logger
}

func (s SoftcomTelemetry) UseMongoMonitor() *event.CommandMonitor {
	return apmmongo.CommandMonitor()
}

func (s SoftcomTelemetry) UseInterceptedGRPCClient(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {

	if opts == nil {
		opts = make([]grpc.DialOption, 0)
	}

	opts = append(opts,
		grpc.WithUnaryInterceptor(apmgrpc.NewUnaryClientInterceptor()),
		grpc.WithStreamInterceptor(apmgrpc.NewStreamClientInterceptor()))

	return grpc.Dial(target, opts...)
}

func (s SoftcomTelemetry) WithContext(ctx context.Context) *logrus.Entry {
	labels := make(map[string]string)
	tx := apm.TransactionFromContext(ctx)
	if tx != nil {
		traceContext := tx.TraceContext()
		labels["trace.id"] = traceContext.Trace.String()
		labels["transaction.id"] = traceContext.Span.String()
		if span := apm.SpanFromContext(ctx); span != nil {
			labels["span.id"] = span.TraceContext().Span.String()
		}
	}

	return s.Logger.WithContext(ctx)
}

func (s SoftcomTelemetry) Publish(topic string, data []byte) error {
	fields := logrus.Fields{
		"topic":   topic,
		"data":    string(data),
		"service": s.serviceName,
	}
	if s.EventStore == nil {
		s.WithFields(fields).Error("cannot publish events because event store not provided in telemetry chain.")
		return nil
	}

	if err := s.EventStore.Publish(topic, data); err != nil {
		s.WithError(err).WithFields(fields).Error("failed to publish event to topic %s", topic)
		return err
	}

	s.WithFields(fields).Infof("published event to %s topic", topic)
	return nil
}

func NewServerTelemetry(serviceName, environment string, opt ...Option) *SoftcomTelemetry {
	serverOptions := mergeServerOptions(opt...)

	apmEnvKey := "ELASTIC_APM_ENVIRONMENT"
	if _, ok := os.LookupEnv(apmEnvKey); !ok {
		_ = os.Setenv(apmEnvKey, environment)
	}

	apmSnKey := "ELASTIC_APM_SERVICE_NAME"
	if _, ok := os.LookupEnv(apmSnKey); !ok {
		_ = os.Setenv(apmSnKey, serviceName)
	}

	logger := logrus.New()
	logger.SetFormatter(serverOptions.formatter)
	logger.ReportCaller = true

	for _, hook := range serverOptions.loggerHooks {
		logger.AddHook(hook)
	}

	if serverOptions.logFile != nil {
		logger.SetOutput(serverOptions.logFile)
	}

	entry := logrus.NewEntry(logger)

	serverOptions.grpcServerOptions = append(serverOptions.grpcServerOptions,
		grpc.StreamInterceptor(grpcmiddleware.ChainStreamServer(
			grpcctxtags.StreamServerInterceptor(),
			grpclogrus.StreamServerInterceptor(entry),
			grpcrecovery.StreamServerInterceptor(),
			apmgrpc.NewStreamServerInterceptor(apmgrpc.WithRecovery()),
			grpclogrus.PayloadStreamServerInterceptor(entry, func(ctx context.Context, fullMethodName string, servingObject interface{}) bool {
				return true
			}),

		)),
		grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(
			grpcctxtags.UnaryServerInterceptor(),
			grpclogrus.UnaryServerInterceptor(entry),
			grpcrecovery.UnaryServerInterceptor(),
			apmgrpc.NewUnaryServerInterceptor(apmgrpc.WithRecovery()),
			grpclogrus.PayloadUnaryServerInterceptor(entry, func(ctx context.Context, fullMethodName string, servingObject interface{}) bool {
				return true
			}),
		)))

	grpcServer := grpc.NewServer(serverOptions.grpcServerOptions...)

	return &SoftcomTelemetry{
		serviceName: serviceName,
		grpcServer:  grpcServer,
		Logger:      logger,
		EventStore:  serverOptions.eventStore,
	}
}
