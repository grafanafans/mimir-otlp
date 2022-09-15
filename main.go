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
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/metric/export/aggregation"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	simple "go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func initMeter() (*controller.Controller, error) {
	ep, err := otlpmetrichttp.New(
		context.Background(),
		otlpmetrichttp.WithEndpoint("mimir:8080"),
		otlpmetrichttp.WithURLPath("/otlp/v1/metrics"),
		otlpmetrichttp.WithInsecure(),
		otlpmetrichttp.WithHeaders(map[string]string{
			"X-Scope-OrgID": "demo",
		}),
	)
	if err != nil {
		return nil, err
	}

	c := controller.New(
		processor.NewFactory(
			simple.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries([]float64{0.05, 0.1, 0.25, 0.5, 1, 2}),
			),
			aggregation.CumulativeTemporalitySelector(),
			processor.WithMemory(true),
		),
		controller.WithExporter(ep),
		controller.WithCollectPeriod(15*time.Second),
	)

	global.SetMeterProvider(c)
	return c, c.Start(context.Background())
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

		if rand.Intn(100) < 10 {
			time.Sleep(200 * time.Millisecond)
		}

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
	c, err := initMeter()
	if err != nil {
		log.Fatal("init meter with error : " + err.Error())
	}
	defer c.Stop(context.Background())

	r := gin.New()
	r.Use(timeDuration())
	r.GET("/users/:id", func(ctx *gin.Context) {
		id := ctx.Param("id")
		ctx.JSON(200, gin.H{
			"id": id,
		})
	})
	r.Run(":8080")
}