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

package storage

import (
	"context"
	"flag"
	"os"
	"strconv"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"go.uber.org/zap"

	"github.com/raft-tech/konfirm-inspections/pkg/storage"
	"github.com/raft-tech/konfirm-inspections/pkg/storage/source"

	"github.com/raft-tech/konfirm-inspections/internal/logging"
)

const (
	MetricsNamespaceEnv     = "METRICS_NAMESPACE"
	MetricsSubsystemEnv     = "METRICS_SUBSYSTEM"
	MetricsJobEnv           = "METRICS_JOB"
	MetricsInstanceEnv      = "METRICS_INSTANCE"
	DefaultMetricsNamespace = "konfirm"
	DefaultMetricsSubsystem = "storage"
	DefaultMetricsJob       = "konfirm_inspections"
	VolumeLabel             = "volume"
	EntryLabel              = "entry"
)

var logger *zap.Logger
var pushGatewayAddr string
var baseDir string
var maxInstances int
var scrub bool
var reads *prometheus.GaugeVec
var readErrors *prometheus.GaugeVec
var writes *prometheus.GaugeVec
var writeErrors *prometheus.GaugeVec
var availableBytes prometheus.Gauge
var totalBytes prometheus.Gauge

func init() {
	logging.RegisterGoFlags(flag.CommandLine)
	flags := flag.CommandLine
	flags.StringVar(&baseDir, "konfirm.base-dir", "", "set the directory used for storage inspections")
	flags.StringVar(&pushGatewayAddr, "konfirm.metrics-gateway", "", "enable pushing metrics to the specified gateway")
	flags.IntVar(&maxInstances, "konfirm.max-instances", 3, "set the maximum number of instances (default is 3)")
	flags.BoolVar(&scrub, "konfirm.scrub", false, "remove any files in the volume that are not in the index")
}

func TestStorage(t *testing.T) {

	logger = logging.NewLogger(GinkgoWriter)
	setupMetrics()

	RegisterTestingT(t)
	Expect(baseDir).To(BeADirectory(), "konfirm.base-dir must be an existing directory")

	RegisterFailHandler(Fail)
	suiteCfg, reporterCfg := GinkgoConfiguration()
	RunSpecs(t, "Storage", suiteCfg, reporterCfg)
}

type test struct {
	name   string
	source source.Source
}

var _ = Describe("Read/Write", func() {

	var volume storage.Volume
	var tests []test

	It("writes data", func() {
		inst, err := volume.NewInstance()
		Expect(err).NotTo(HaveOccurred())
		hadError := false
		for _, t := range tests {
			labels := prometheus.Labels{
				VolumeLabel: baseDir,
				EntryLabel:  t.name,
			}
			entry := inst.Add(labels[EntryLabel], t.source)
			writes.With(labels).Set(float64((*entry.WriteDuration).Milliseconds()))
			if entry.Error != nil {
				writeErrors.With(labels).Set(1.0)
				hadError = true
			}
		}
		Expect(hadError).NotTo(BeTrue(), "one or more write errors occurred")
	})

	It("reads data", func() {
		measurements := make(map[string]*observer)
		hadErr := false
		for _, instance := range volume.Instances() {
			entries, err := instance.Walk()
			Expect(err).NotTo(HaveOccurred())
			for i := range entries {
				var obs *observer
				if o, ok := measurements[entries[i].Path]; ok {
					obs = o
				} else {
					obs = &observer{}
					measurements[entries[i].Path] = obs
				}
				if d := entries[i].ReadDuration; d != nil {
					obs.Observe(*d)
				}
				obs.ObserveError(entries[i].Error)
				if entries[i].Error != nil {
					hadErr = true
				}
			}
		}
		for path, obs := range measurements {
			l := prometheus.Labels{
				VolumeLabel: baseDir,
				EntryLabel:  path,
			}
			reads.With(l).Set(float64(obs.Average().Milliseconds()))
			readErrors.With(l).Set(float64(obs.errors))
		}
		Expect(hadErr).ToNot(BeTrue(), "one or more read errors occurred")
	})

	BeforeAll(func() {

		logger.Info("starting storage inspections", zap.String("baseDir", baseDir))

		var err error
		volume, err = storage.New(baseDir, storage.WithMaxInstances(maxInstances), storage.WithLogger(logger))
		Expect(err).NotTo(HaveOccurred())

		if scrub {
			logger.Debug("starting scrub")
			if e := volume.Scrub(); e == nil {
				logger.Info("completed scrub")
			} else {
				logger.Error("error scrubbing volume", zap.Error(e))
				Expect(e).NotTo(HaveOccurred())
			}
		}

		args := flag.CommandLine.Args()
		if len(args) == 0 {
			tests = []test{
				{
					name:   "512MiB",
					source: source.New(512 * source.Megabyte),
				},
			}
		} else {

			// Specs are defined in the format NAME:SIZE where SIZE is in the format [N][unit]
			// N being an integer and unit being one of KiB, MiB, GiB.
			// For example, Medium:512MiB would create a spec named "Medium" with a 512MiB Source.
			for i := range args {

				spec := strings.SplitN(args[i], ":", 2)
				if len(spec) != 2 {
					msg := "malformed test spec"
					logger.Warn(msg, zap.String("spec", args[i]))
					Fail(msg)
				}

				t := test{
					name: spec[0],
				}

				var size, unit string
				if l := len(spec[1]); l >= 4 {
					size = spec[1][:l-3]
					unit = spec[1][l-3:]
				} else {
					msg := "malformed test size"
					logger.Warn(msg, zap.String("size", unit))
					Fail(msg)
				}

				var n int
				if num, err := strconv.Atoi(size); err == nil {
					n = num
				} else {
					msg := "invalid spec size"
					logger.Warn(msg, zap.String("size", size))
					Fail(msg)
				}

				switch unit {
				case "KiB":
					t.source = source.New(n * source.Kilobyte)
				case "MiB":
					t.source = source.New(n * source.Megabyte)
				case "GiB":
					t.source = source.New(n * source.Gigabyte)
				default:
					msg := "invalid source unit"
					logger.Warn(msg, zap.String("unit", unit))
					Fail(msg)
				}
				tests = append(tests, t)
			}
		}
	})

}, Ordered)

func setupMetrics() {

	var namespace = DefaultMetricsNamespace
	if ns, ok := os.LookupEnv(MetricsNamespaceEnv); ok {
		namespace = ns
	}

	var subsystem = DefaultMetricsSubsystem
	if s, ok := os.LookupEnv(MetricsSubsystemEnv); ok {
		subsystem = s
	}

	reads = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "avg_read_duration",
	}, []string{VolumeLabel, EntryLabel})

	readErrors = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "read_errors",
	}, []string{VolumeLabel, EntryLabel})

	writes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "write_duration",
	}, []string{VolumeLabel, EntryLabel})

	writeErrors = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "write_error",
	}, []string{VolumeLabel, EntryLabel})

	availableBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "available_bytes",
		ConstLabels: prometheus.Labels{
			"directory": baseDir,
		},
	})

	totalBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "total_bytes",
		ConstLabels: prometheus.Labels{
			"directory": baseDir,
		},
	})
}

var _ = AfterSuite(func(ctx context.Context) {

	svc := strings.TrimSpace(pushGatewayAddr)
	if svc == "" {
		return
	}

	jobName := DefaultMetricsJob
	if j, ok := os.LookupEnv(MetricsJobEnv); ok {
		jobName = j
	}

	instance := "unknown"
	if i, ok := os.LookupEnv(MetricsInstanceEnv); ok {
		instance = i
	} else if host, err := os.Hostname(); err == nil {
		instance = host
	}

	logger := logger.With(zap.String("gateway", svc))
	logger.Debug("initializing metrics gateway", zap.String("job", jobName), zap.String("instance", instance))

	pusher := push.New(svc, jobName)
	pusher.Grouping("instance", instance)

	pusher.Collector(reads)
	pusher.Collector(readErrors)
	pusher.Collector(writes)
	pusher.Collector(writeErrors)

	if d, e := storage.GetDisk(baseDir); e == nil {
		pusher.Collector(availableBytes)
		availableBytes.Set(float64(d.AvailableBytes()))
		pusher.Collector(totalBytes)
		totalBytes.Set(float64(d.TotalBytes()))
	}

	logger.Debug("pushing metrics")
	if err := pusher.PushContext(ctx); err == nil {
		logger.Info("successfully pushed metrics")
	} else {
		logger.Error("error pushing metrics", zap.Error(err))
	}
})
