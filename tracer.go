package tracer

import (
	"context"
	"io"
	"net/http"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/reverted/ex"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

type Logger interface {
	Fatal(...interface{})
}

func New(logger Logger) *tracer {

	cfg, err := config.FromEnv()
	if err != nil {
		logger.Fatal(err)
	}

	sampler, err := jaeger.NewProbabilisticSampler(1.0)
	if err != nil {
		logger.Fatal(err)
	}

	tr, closer, err := cfg.NewTracer(config.Sampler(sampler))
	if err != nil {
		logger.Fatal(err)
	}

	return &tracer{tr, closer}
}

type tracer struct {
	opentracing.Tracer
	io.Closer
}

func (self *tracer) StartSpan(ctx context.Context, name string, tags ...ex.SpanTag) (ex.Span, context.Context) {
	var opts []opentracing.StartSpanOption

	for _, tag := range tags {
		opts = append(opts, opentracing.Tag{tag.Key, tag.Value})
	}

	return opentracing.StartSpanFromContextWithTracer(ctx, self.Tracer, name, opts...)
}

func (self *tracer) InjectSpan(ctx context.Context, r *http.Request) {

	span := opentracing.SpanFromContext(ctx)

	ext.SpanKindRPCClient.Set(span)

	ext.HTTPUrl.Set(span, r.URL.String())
	ext.HTTPMethod.Set(span, r.Method)

	self.Tracer.Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header),
	)
}

func (self *tracer) ExtractSpan(r *http.Request, name string) (ex.Span, context.Context) {

	spanCtx, _ := self.Tracer.Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header),
	)

	return opentracing.StartSpanFromContextWithTracer(r.Context(), self.Tracer, name, ext.RPCServerOption(spanCtx))
}
