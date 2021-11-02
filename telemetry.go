package gotelemetry

import (
	"errors"
	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpclogrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpcctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpcopentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"github.com/sirupsen/logrus"
	"github.com/softcomoss/jetstreamclient"
	"go.elastic.co/apm/module/apmgrpc"
	"go.elastic.co/ecslogrus"
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
	grpcServer *grpc.Server
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

func (s SoftcomTelemetry) UseInterceptedGRPCClient(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {

	if opts == nil {
		opts = make([]grpc.DialOption, 0)
	}

	opts = append(opts,
		grpc.WithUnaryInterceptor(apmgrpc.NewUnaryClientInterceptor()),
		grpc.WithStreamInterceptor(apmgrpc.NewStreamClientInterceptor()))

	return grpc.Dial(target, opts...)
}

func (s SoftcomTelemetry) Publish(topic string, data []byte) error {
	s.WithFields(logrus.Fields{
		"topic":   topic,
		"data":    data,
		"service": s.GetServiceName(),
	}).Infof("published event to %s topic", topic)

	return s.EventStore.Publish(topic, data)
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
			grpcopentracing.StreamServerInterceptor(),
			grpclogrus.StreamServerInterceptor(entry),
			grpcrecovery.StreamServerInterceptor(),
			apmgrpc.NewStreamServerInterceptor(apmgrpc.WithRecovery()),

		)),
		grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(
			grpcctxtags.UnaryServerInterceptor(),
			grpcopentracing.UnaryServerInterceptor(),
			grpclogrus.UnaryServerInterceptor(entry),
			apmgrpc.NewUnaryServerInterceptor(apmgrpc.WithRecovery()),
			grpcrecovery.UnaryServerInterceptor(),
		)))

	grpcServer := grpc.NewServer(serverOptions.grpcServerOptions...)

	return &SoftcomTelemetry{
		grpcServer: grpcServer,
		Logger:     logger,
		EventStore: serverOptions.eventStore,
	}
}
