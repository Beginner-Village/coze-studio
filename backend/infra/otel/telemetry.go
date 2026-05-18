/*
 * Copyright 2025 ynet-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package otel

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ynet-dev/ynet-studio/backend/pkg/logs"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	envEnable       = "YNET_LOOP_TELEMETRY_ENABLE"
	envEndpoint     = "YNET_LOOP_TELEMETRY_ENDPOINT"
	envWorkspace    = "YNET_LOOP_WORKSPACE_ID"
	envToken        = "YNET_LOOP_TELEMETRY_TOKEN"
	envSamplerRatio = "YNET_LOOP_TRACE_RATIO"
	envServiceName  = "YNET_SERVICE_NAME"
	defaultSvcName  = "ynet-studio"
	defaultSample   = 1.0
	defaultTimeout  = 5 * time.Second

	wsAttrKey = "cozeloop.workspace_id"
)

// Init configures the global OTEL TracerProvider, returning a shutdown function when enabled.
func Init(ctx context.Context) (func(context.Context) error, error) {
	// Private-deployment default: telemetry ON unless explicitly disabled.
	// In open-source release this stays opt-in (envEnable unset → off in upstream),
	// but for ynet-studio we want trace ingestion working out of the box.
	enabled := strings.ToLower(strings.TrimSpace(os.Getenv(envEnable)))
	if enabled == "0" || enabled == "false" {
		logs.Infof("otel: telemetry disabled (%s=%s)", envEnable, enabled)
		return nil, nil
	}

	endpoint := strings.TrimSpace(os.Getenv(envEndpoint))
	if endpoint == "" {
		// Private-deployment default: Loop runs in the same namespace,
		// expose its OTel ingest endpoint via Service DNS.
		endpoint = "http://ynet-loop-app:8888/v1/loop/opentelemetry/v1/traces"
		logs.Infof("otel: %s not set, defaulting to %s", envEndpoint, endpoint)
	}
	fallbackWS := strings.TrimSpace(os.Getenv(envWorkspace))
	token := strings.TrimSpace(os.Getenv(envToken))
	if token == "" {
		// Loop OtelIngestTraces does not require a bearer token in the
		// private deployment (see backend/api/router/.../coze.loop.apis.go:
		// _otelingesttracesMw returns nil). Send a placeholder so existing
		// header construction still works.
		token = "private"
	}

	exporter := &dynamicWSExporter{
		endpoint:   endpoint,
		token:      token,
		fallbackWS: fallbackWS,
		exporters:  make(map[string]sdktrace.SpanExporter),
	}

	res, err := buildResource(ctx)
	if err != nil {
		return nil, err
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(readRatio()))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(exporter)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	logs.Infof("otel: telemetry initialized, exporting to %s (dynamic workspace routing enabled)", endpoint)

	return func(shutdownCtx context.Context) error {
		ctxWithTimeout, cancel := context.WithTimeout(shutdownCtx, defaultTimeout)
		defer cancel()
		shutdownErr := tp.Shutdown(ctxWithTimeout)
		exporter.Shutdown(ctxWithTimeout)
		return shutdownErr
	}, nil
}

// dynamicWSExporter routes spans to the correct workspace by reading
// the "cozeloop.workspace_id" span attribute and creating per-workspace OTLP exporters.
type dynamicWSExporter struct {
	mu         sync.Mutex
	exporters  map[string]sdktrace.SpanExporter
	endpoint   string
	token      string
	fallbackWS string
}

func (d *dynamicWSExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if len(spans) > 0 {
		logs.Infof("otel: ExportSpans called with %d spans", len(spans))
		for i, s := range spans {
			if i < 3 { // log first 3 spans for debugging
				logs.Infof("otel:   span[%d] name=%s ws=%s", i, s.Name(), d.extractWorkspaceID(s))
			}
		}
	}

	grouped := make(map[string][]sdktrace.ReadOnlySpan)
	for _, s := range spans {
		ws := d.extractWorkspaceID(s)
		grouped[ws] = append(grouped[ws], s)
	}

	var errs []error
	for ws, batch := range grouped {
		exp, err := d.getOrCreateExporter(ctx, ws)
		if err != nil {
			logs.Warnf("otel: failed to create exporter for workspace %s: %v", ws, err)
			errs = append(errs, err)
			continue
		}
		logs.Infof("otel: exporting %d spans to workspace %s", len(batch), ws)
		if err := exp.ExportSpans(ctx, batch); err != nil {
			if strings.Contains(err.Error(), "invalid UTF-8") {
				// Retry with sanitized spans: export one by one, skip bad ones
				logs.Warnf("otel: UTF-8 error, retrying spans individually for workspace %s", ws)
				for _, s := range batch {
					if err2 := exp.ExportSpans(ctx, []sdktrace.ReadOnlySpan{s}); err2 != nil {
						logs.Warnf("otel: skipping span %s due to: %v", s.Name(), err2)
					} else {
						logs.Infof("otel: exported span %s individually", s.Name())
					}
				}
			} else {
				logs.Warnf("otel: failed to export spans for workspace %s: %v", ws, err)
				errs = append(errs, err)
			}
		} else {
			logs.Infof("otel: successfully exported %d spans to workspace %s", len(batch), ws)
		}
	}
	return errors.Join(errs...)
}

func (d *dynamicWSExporter) Shutdown(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	var errs []error
	for ws, exp := range d.exporters {
		if err := exp.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
		delete(d.exporters, ws)
	}
	return errors.Join(errs...)
}

func (d *dynamicWSExporter) extractWorkspaceID(s sdktrace.ReadOnlySpan) string {
	// Private deployment: always use the configured Loop workspace ID
	// to ensure spans are routed to the correct Loop workspace
	if d.fallbackWS != "" {
		return d.fallbackWS
	}
	for _, attr := range s.Attributes() {
		if string(attr.Key) == wsAttrKey {
			v := attr.Value.AsString()
			if v != "" && v != "0" {
				return v
			}
		}
	}
	return d.fallbackWS
}

func (d *dynamicWSExporter) getOrCreateExporter(ctx context.Context, ws string) (sdktrace.SpanExporter, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if exp, ok := d.exporters[ws]; ok {
		return exp, nil
	}
	exp, err := buildExporter(ctx, d.endpoint, ws, d.token)
	if err != nil {
		return nil, err
	}
	d.exporters[ws] = exp
	logs.Infof("otel: created exporter for workspace %s", ws)
	return exp, nil
}

func buildExporter(ctx context.Context, endpoint, workspace, token string) (sdktrace.SpanExporter, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"cozeloop-workspace-id": workspace,
	}
	if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = "Bearer " + token
	}
	headers["authorization"] = token

	opts := []otlptracehttp.Option{
		otlptracehttp.WithHeaders(headers),
		otlptracehttp.WithTimeout(defaultTimeout),
	}

	if u.Scheme == "http" {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if host := u.Host; host != "" {
		opts = append(opts, otlptracehttp.WithEndpoint(host))
	}

	if trimmed := strings.TrimSpace(u.Path); trimmed != "" {
		opts = append(opts, otlptracehttp.WithURLPath("/"+strings.TrimPrefix(trimmed, "/")))
	}

	return otlptracehttp.New(ctx, opts...)
}

func buildResource(ctx context.Context) (*resource.Resource, error) {
	svcName := strings.TrimSpace(os.Getenv(envServiceName))
	if svcName == "" {
		svcName = defaultSvcName
	}

	appEnv := strings.TrimSpace(os.Getenv("APP_ENV"))

	return resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(svcName),
			attribute.String("environment", appEnv),
		),
	)
}

func readRatio() float64 {
	val := strings.TrimSpace(os.Getenv(envSamplerRatio))
	if val == "" {
		return defaultSample
	}

	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		logs.Warnf("otel: invalid %s=%s, fallback to %f", envSamplerRatio, val, defaultSample)
		return defaultSample
	}
	if parsed <= 0 {
		return 0
	}
	if parsed > 1 {
		return 1
	}
	return parsed
}
