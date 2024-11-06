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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/raft-tech/konfirm-inspections/inspections"
	"github.com/raft-tech/konfirm-inspections/internal/logging"
	"github.com/raft-tech/konfirm-inspections/pkg/storage"
	"github.com/raft-tech/konfirm-inspections/pkg/storage/source"
)

const (
	VolumeLabel = "volume"
	EntryLabel  = "entry"
)

var (
	logger *zap.Logger

	baseDir      string
	maxInstances int
	scrub        bool

	metrics        inspections.Metrics
	reads          *prometheus.GaugeVec
	readErrors     *prometheus.GaugeVec
	writes         *prometheus.GaugeVec
	writeErrors    *prometheus.GaugeVec
	availableBytes prometheus.Gauge
	totalBytes     prometheus.Gauge
)

func init() {
	flags := flag.CommandLine
	inspections.RegisterTestFlags(flags)
	flags.StringVar(&baseDir, "konfirm.base-dir", "", "set the directory used for storage inspections")
	flags.IntVar(&maxInstances, "konfirm.max-instances", 3, "set the maximum number of instances (default is 3)")
	flags.BoolVar(&scrub, "konfirm.scrub", false, "remove any files in the volume that are not in the index")
}

func TestStorage(t *testing.T) {

	logger = logging.NewLogger(GinkgoWriter)
	ctx, done := context.WithCancel(logging.NewContext(context.Background(), logger.Named("healthz")))
	defer done()
	inspections.StartHealthz(ctx)
	setupMetrics()

	RegisterTestingT(t)
	RegisterFailHandler(Fail)
	g := NewGomegaWithT(t)
	g.Expect(baseDir).To(BeADirectory(), "konfirm.base-dir must be an existing directory")

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
					name:   "512Mi",
					source: source.New(512 * 1024 * 1024),
				},
			}
		} else {

			// Specs are defined in the format NAME:SIZE where SIZE is in the format [N][unit]
			// N being an integer and unit being one of Ki, Mi, Gi.
			// For example, Medium:512Mi would create a spec named "Medium" with a 512 mebibyte Source.
			for i := range args {
				if t, err := source.NewSpec(args[i], ""); err == nil {
					tests = append(tests, test{name: t.Name(), source: t.Generate()})
				} else {
					logger.Error("malformed test spec", zap.Error(err))
				}
			}
		}
	})

}, Ordered)

func setupMetrics() {

	metrics = inspections.NewMetrics()
	const namespace = inspections.MetricsNamespace
	const subsystem = "storage"

	reads = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "avg_read_duration_ms",
	}, []string{VolumeLabel, EntryLabel})
	metrics.Register(reads)

	readErrors = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "read_errors",
	}, []string{VolumeLabel, EntryLabel})
	metrics.Register(readErrors)

	writes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "write_duration_ms",
	}, []string{VolumeLabel, EntryLabel})
	metrics.Register(writes)

	writeErrors = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "write_error",
	}, []string{VolumeLabel, EntryLabel})
	metrics.Register(writeErrors)

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
	if d, e := storage.GetDisk(baseDir); e == nil {
		availableBytes.Set(float64(d.AvailableBytes()))
		metrics.Register(availableBytes)
		totalBytes.Set(float64(d.TotalBytes()))
		metrics.Register(totalBytes)
	}
	metrics.Push(ctx)
})
