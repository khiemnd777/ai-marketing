package telemetry

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const traceMetadataKey = "studioTrace"

type RiverTracing struct{ river.MiddlewareDefaults }

func RiverPlugins() []rivertype.Plugin { return []rivertype.Plugin{&RiverTracing{}} }

func (middleware *RiverTracing) InsertMany(ctx context.Context, params []*rivertype.JobInsertParams, inner func(context.Context) ([]*rivertype.JobInsertResult, error)) ([]*rivertype.JobInsertResult, error) {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	if carrier.Get("traceparent") == "" {
		return inner(ctx)
	}
	for _, item := range params {
		metadata := map[string]any{}
		_ = json.Unmarshal(item.Metadata, &metadata)
		metadata[traceMetadataKey] = map[string]string{"traceparent": carrier.Get("traceparent"), "tracestate": carrier.Get("tracestate")}
		if raw, err := json.Marshal(metadata); err == nil {
			item.Metadata = raw
		}
	}
	return inner(ctx)
}

func (middleware *RiverTracing) Work(ctx context.Context, job *rivertype.JobRow, inner func(context.Context) error) error {
	metadata := map[string]any{}
	if json.Unmarshal(job.Metadata, &metadata) == nil {
		if raw, ok := metadata[traceMetadataKey].(map[string]any); ok {
			carrier := propagation.MapCarrier{}
			if value, ok := raw["traceparent"].(string); ok {
				carrier.Set("traceparent", value)
			}
			if value, ok := raw["tracestate"].(string); ok {
				carrier.Set("tracestate", value)
			}
			ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
		}
	}
	ctx, span := otel.Tracer("studio-worker/river").Start(ctx, "river "+job.Kind, trace.WithSpanKind(trace.SpanKindConsumer), trace.WithAttributes(
		attribute.String("messaging.system", "river"),
		attribute.String("messaging.operation.type", "process"),
		attribute.String("messaging.destination.name", job.Queue),
		attribute.String("messaging.message.id", strconv.FormatInt(job.ID, 10)),
		attribute.String("river.job.kind", job.Kind),
		attribute.Int("river.job.attempt", job.Attempt),
	))
	defer span.End()
	err := inner(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "job failed")
	}
	return err
}

var (
	_ rivertype.JobInsertMiddleware = (*RiverTracing)(nil)
	_ rivertype.WorkerMiddleware    = (*RiverTracing)(nil)
)
