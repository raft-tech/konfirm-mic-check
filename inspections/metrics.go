/*
 * Copyright (c) 2024 Raft, LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package inspections

import (
	"context"
	"flag"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/raft-tech/konfirm-inspections/internal/healthz"
	"github.com/raft-tech/konfirm-inspections/internal/logging"
)

const (
	MetricsNamespace        = "konfirm"
	MetricsGatewayFlag      = "metrics-gateway"
	MetricsJobFlag          = "metrics-job"
	MetricsInstanceFlag     = "metrics-instance"
	TestMetricsGatewayFlag  = "konfirm.metrics-gateway"
	TestMetricsJobFlag      = "konfirm.metrics-job"
	TestMetricsInstanceFlag = "konfirm.metrics-instance"
)

var (
	probeAddr       = ""
	metricsGateway  = ""
	metricsJob      = "konfirm_inspections"
	metricsInstance = ""
)

func RegisterCmdFlags(set *pflag.FlagSet) {
	logging.RegisterCmdFlags(set)
	set.StringVar(&probeAddr, healthz.ListenFlag, probeAddr, "sets the listen addr for health probe server")
	set.StringVar(&metricsGateway, MetricsGatewayFlag, metricsGateway, "push metrics to the specified host")
	set.StringVar(&metricsJob, MetricsJobFlag, metricsJob, "specify the metrics job")
	set.StringVar(&metricsInstance, MetricsInstanceFlag, metricsInstance, "specify the metrics instance (for grouping)")
}

func RegisterTestFlags(set *flag.FlagSet) {
	logging.RegisterTestFlags(set)
	set.StringVar(&probeAddr, healthz.ListenFlag, probeAddr, "sets the listen addr for health probe server")
	set.StringVar(&metricsGateway, TestMetricsGatewayFlag, metricsGateway, "push metrics to the specified host")
	set.StringVar(&metricsJob, TestMetricsJobFlag, metricsJob, "specify the metrics job")
	set.StringVar(&metricsInstance, TestMetricsInstanceFlag, metricsInstance, "specify the metrics instance (for grouping)")
}

func StartHealthz(ctx context.Context) {

	if probeAddr == "" {
		return
	}

	logger := logging.FromContext(ctx)
	if probes, err := healthz.ListenAndServe(ctx, probeAddr, func(err error) {
		logger.Error("error serving http probes", zap.Error(err))
	}); err == nil {
		probes.Ready(true)
	} else {
		logger.Error("error starting http probe server", zap.Error(err))
	}
}

type Metrics interface {
	Register(prometheus.Collector)
	Push(context.Context)
}

type nopMetrics struct{}

func (_ nopMetrics) Register(_ prometheus.Collector) {}
func (_ nopMetrics) Push(_ context.Context)          {}

type metrics struct {
	*push.Pusher
}

func NewMetrics() Metrics {

	if metricsGateway == "" {
		return nopMetrics{}
	}

	m := &metrics{
		Pusher: push.New(metricsGateway, metricsJob),
	}

	i := metricsInstance
	if i == "" {
		if host, err := os.Hostname(); err == nil {
			i = host
		} else {
			i = "unknown"
		}
	}
	m.Grouping("instance", i)
	return m
}

func (m *metrics) Register(collector prometheus.Collector) {
	m.Pusher.Collector(collector)
}

func (m *metrics) Push(ctx context.Context) {

	if m == nil {
		logging.FromContext(ctx).Debug("skipping metrics: gateway not defined")
		return
	}

	logger := logging.FromContext(ctx).
		With(
			zap.String("gateway", metricsGateway),
			zap.String("job", metricsJob),
			zap.String("instance", metricsInstance),
		)

	logger.Debug("pushing metrics")
	if err := m.Pusher.PushContext(ctx); err == nil {
		logger.Info("successfully pushed metrics")
	} else {
		logger.Error("error pushing metrics", zap.Error(err))
	}
}
