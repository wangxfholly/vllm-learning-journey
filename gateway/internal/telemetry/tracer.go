package telemetry

import (
"context"
"log"
"os"
"time"

"go.opentelemetry.io/otel"
"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
"go.opentelemetry.io/otel/sdk/resource"
sdktrace "go.opentelemetry.io/otel/sdk/trace"
semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
"go.opentelemetry.io/otel/trace"
)

const serviceName = "vllm-gateway"

var Tracer trace.Tracer

func InitTracer(ctx context.Context) (func(context.Context) error, error) {
var exporter sdktrace.SpanExporter
var err error

endpoint := os.Getenv("OTEL_OTLP_ENDPOINT")
if endpoint == "" {
exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
log.Println("[telemetry] Trace 导出器: stdout(控制台)")
} else {
exporter, err = otlptracegrpc.New(ctx,
otlptracegrpc.WithEndpoint(endpoint),
otlptracegrpc.WithInsecure(),
)
log.Printf("[telemetry] Trace 导出器: OTLP -> %s\n", endpoint)
}
if err != nil {
return nil, err
}

res, err := resource.New(ctx,
resource.WithAttributes(semconv.ServiceName(serviceName)),
)
if err != nil {
return nil, err
}

tp := sdktrace.NewTracerProvider(
sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(time.Second)),
sdktrace.WithResource(res),
sdktrace.WithSampler(sdktrace.AlwaysSample()),
)
otel.SetTracerProvider(tp)
Tracer = tp.Tracer(serviceName)

return tp.Shutdown, nil
}
