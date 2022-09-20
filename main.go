// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	promExporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/sdk/metric"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func initMeter() *metric.MeterProvider {

	// init otlpmetrichttp exporter
	ep, _ := otlpmetrichttp.New(
		context.Background(),
		otlpmetrichttp.WithEndpoint("mimir:8080"),
		otlpmetrichttp.WithURLPath("/otlp/v1/metrics"),
		otlpmetrichttp.WithInsecure(),
		otlpmetrichttp.WithHeaders(map[string]string{
			"X-Scope-OrgID": "demo",
		}),
	)

	// init stdout exporter
	ep2, _ := stdoutmetric.New()

	// init prometheus exporter
	ep3 := promExporter.New()
	prometheus.Register(ep3.Collector)

	provider := metric.NewMeterProvider(
		metric.WithReader(
			metric.NewPeriodicReader(ep, metric.WithInterval(15*time.Second)),
		),
		metric.WithReader(
			metric.NewPeriodicReader(ep2, metric.WithInterval(15*time.Second)),
		),
		metric.WithReader(ep3),
	)
	global.SetMeterProvider(provider)

	return provider
}

func timeDuration() func(ctx *gin.Context) {
	meter := global.Meter("gin-otlp-meter")
	httpDurationsHistogram, _ := meter.SyncFloat64().Histogram(
		"http_durations_histogram_seconds",
		instrument.WithDescription("Http latency distributions."),
	)

	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()
		status := strconv.Itoa(ctx.Writer.Status())
		method := ctx.Request.Method
		elapsed := float64(time.Since(start)) / float64(time.Second)
		httpDurationsHistogram.Record(
			ctx,
			elapsed,
			attribute.String("method", method),
			attribute.String("status", status),
		)
	}
}

func main() {
	provider := initMeter()
	defer provider.Shutdown(context.Background())

	r := gin.New()
	r.Use(timeDuration())
	r.GET("/users/:id", func(ctx *gin.Context) {
		id := ctx.Param("id")
		// 模拟随机延迟请求
		if rand.Intn(100) < 10 {
			time.Sleep(200 * time.Millisecond)
		}
		ctx.JSON(200, gin.H{
			"id": id,
		})
	})

	// register prometheus handler
	promHandler := promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	)
	r.GET("/metrics", func(ctx *gin.Context) {
		promHandler.ServeHTTP(ctx.Writer, ctx.Request)
	})

	r.Run(":8080")
}
