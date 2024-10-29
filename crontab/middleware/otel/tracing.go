package otel

import (
	"context"

	"github.com/flc1125/go-cron/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var scopeName = "github.com/go-kratos-ecosystem/components/v2/crontab/middleware/otel"

var (
	attrJobName     = attribute.Key("job.name")
	attrJobID       = attribute.Key("job.id")
	attrJobPrevTime = attribute.Key("job.prev.time")
	attrJobNextTime = attribute.Key("job.next.time")
)

type JobWithName interface {
	cron.Job
	Name() string
}

type options struct {
	tp trace.TracerProvider
}

type Option func(*options)

func newOptions(opts ...Option) *options {
	opt := &options{
		tp: otel.GetTracerProvider(),
	}
	for _, o := range opts {
		o(opt)
	}
	return opt
}

func NewTracing(opts ...Option) cron.Middleware {
	o := newOptions(opts...)

	tracer := o.tp.Tracer(scopeName)
	return func(original cron.Job) cron.Job {
		return cron.JobFunc(func(ctx context.Context) (err error) {
			job, ok := any(original).(JobWithName)
			if !ok {
				return original.Run(ctx)
			}

			// The span is created here, and it will be ended when the job is done.
			var span trace.Span
			ctx, span = tracer.Start(ctx, job.Name(),
				trace.WithSpanKind(trace.SpanKindInternal),
			)
			defer span.End()
			defer func() {
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
				} else {
					span.SetStatus(codes.Ok, "")
				}
			}()

			// Set attributes.
			attrs := []attribute.KeyValue{
				attrJobName.String(job.Name()),
			}
			attrs = append(attrs, entryAttributes(ctx)...)

			span.SetAttributes(attrs...)

			// The job is run here.
			err = job.Run(ctx)
			return
		})
	}
}

func entryAttributes(ctx context.Context) []attribute.KeyValue {
	entry, ok := cron.EntryFromContext(ctx)
	if !ok {
		return nil
	}

	return []attribute.KeyValue{
		attrJobID.Int(int(entry.ID())),
		attrJobPrevTime.String(entry.Prev().String()),
		attrJobNextTime.String(entry.Next().String()),
	}
}
