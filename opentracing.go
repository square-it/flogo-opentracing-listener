package opentracing

import (
	"errors"
	"fmt"
	logger "github.com/project-flogo/core/support/log"
	"io"
	"os"
	"strings"

	"github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin-contrib/zipkin-go-opentracing"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/transport"
)

var (
	opentracinglogger = logger.ChildLogger(logger.RootLogger(), "opentracing")
)

const (
	EnvVarsPrefix        = "FLOGO_OPENTRACING_"
	EnvVarImplementation = EnvVarsPrefix + "IMPLEMENTATION"
	EnvVarTransport      = EnvVarsPrefix + "TRANSPORT"
	EnvVarEndpoints      = EnvVarsPrefix + "ENDPOINTS"
)

const (
	hostPort = "0.0.0.0:0" // not applicable -> leave as-is

	// Debug mode.
	debug = false

	// same span can be set to true for RPC style spans (Zipkin V1) vs Node style (OpenTracing)
	sameSpan = true

	// make Tracer generate 128 bit traceID's for root spans.
	traceID128Bit = true
)

type Config struct {
	Implementation string   `json:"implementation"`
	Transport      string   `json:"transport"`
	Endpoints      []string `json:"endpoints"`
}

func initFromEnvVars() {
	globalOpenTracingImplementation, exists := os.LookupEnv(EnvVarImplementation)
	if !exists {
		return
	}

	opentracinglogger.Infof("Implementation : %s", globalOpenTracingImplementation)

	globalOpenTracingTransport, exists := os.LookupEnv(EnvVarTransport)
	if !exists {
		opentracinglogger.Errorf("Environment variable %s must be set to initialize OpenTracing tracer.", EnvVarTransport)
		return
	}
	opentracinglogger.Infof("Transport      : %s", globalOpenTracingTransport)

	globalOpenTracingEndpoints, exists := os.LookupEnv(EnvVarEndpoints)
	if !exists {
		opentracinglogger.Errorf("Environment variable %s must be set to initialize OpenTracing tracer.", EnvVarEndpoints)
		return
	}
	opentracinglogger.Infof("Endpoints      : %s", globalOpenTracingEndpoints)

	openTracingConfig := &Config{Implementation: globalOpenTracingImplementation, Transport: globalOpenTracingTransport, Endpoints: strings.Split(globalOpenTracingEndpoints, ",")}

	tracer, _ := InitTracer("flogo", openTracingConfig)
	opentracing.SetGlobalTracer(*tracer)

}

func initJaegerStdOut(service string) (*opentracing.Tracer, io.Closer) {
	tracer, closer := jaeger.NewTracer(service, jaeger.NewConstSampler(true), jaeger.NewLoggingReporter(jaeger.StdLogger))

	return &tracer, closer
}

func initJaegerHttp(service string, endpoint string) (*opentracing.Tracer, io.Closer) {
	sender := transport.NewHTTPTransport(endpoint, transport.HTTPBatchSize(1))

	tracer, closer := jaeger.NewTracer(service, jaeger.NewConstSampler(true), jaeger.NewRemoteReporter(sender))

	return &tracer, closer
}

func initZipkinHttp(serviceName string, endpoint string) *opentracing.Tracer {
	// Create our HTTP collector.
	collector, err := zipkin.NewHTTPCollector(endpoint)
	if err != nil {
		panic(fmt.Sprintf("unable to create Zipkin HTTP collector: %+v\n", err))

	}

	// Create our recorder.
	recorder := zipkin.NewRecorder(collector, debug, hostPort, serviceName)

	// Create our tracer.
	tracer, err := zipkin.NewTracer(
		recorder,
		zipkin.ClientServerSameSpan(sameSpan),
		zipkin.TraceID128Bit(traceID128Bit),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to create Zipkin tracer: %+v\n", err))
	}

	return &tracer
}

func initZipkinKafka(serviceName string, endpoint []string) *opentracing.Tracer {
	// Create our Kafka collector.
	collector, err := zipkin.NewKafkaCollector(endpoint)
	//collector, err := zipkin.NewKafkaCollector(endpoint, zipkin.KafkaLogger(zipkin.LogWrapper(log.New(os.Stdout, log.Prefix(), log.Flags()))))

	if err == nil {
		// Create our recorder.
		recorder := zipkin.NewRecorder(collector, debug, hostPort, serviceName)

		// Create our tracer.
		tracer, err := zipkin.NewTracer(
			recorder,
			zipkin.ClientServerSameSpan(sameSpan),
			zipkin.TraceID128Bit(traceID128Bit),
		)
		if err != nil {
			panic(fmt.Sprintf("unable to create Zipkin tracer: %+v\n", err))
		}
		return &tracer
	} else {
		// panic(fmt.Sprintf("unable to create Zipkin Kafka collector: %+v\n", err))
		return nil
	}
}

func InitTracer(serviceName string, openTracingConfig *Config) (*opentracing.Tracer, error) {
	switch openTracingConfig.Implementation {
	case "zipkin":
		switch openTracingConfig.Transport {
		case "http":
			return initZipkinHttp(serviceName, openTracingConfig.Endpoints[0]), nil
		case "kafka":
			return initZipkinKafka(serviceName, openTracingConfig.Endpoints), nil
		default:
			return nil, errors.New("supported transports for OpenTracing Zipkin traecer are 'http' or 'kafka'")
		}
	case "jaeger":
		switch openTracingConfig.Transport {
		case "stdout":
			jaegerTracer, _ := initJaegerStdOut(serviceName)
			return jaegerTracer, nil
		case "http":
			jaegerTracer, _ := initJaegerHttp(serviceName, openTracingConfig.Endpoints[0])
			return jaegerTracer, nil
		default:
			return nil, errors.New("supported transport for OpenTracing Jaeger traecer is 'stdout'")
		}
	default:
		return nil, errors.New("supported implementations for OpenTracing are 'jaeger' or 'zipkin'")
	}
}
