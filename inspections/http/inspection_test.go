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

package http

import (
	"context"
	"flag"
	"io"
	gohttp "net/http"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	ginkgo "github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/raft-tech/konfirm-inspections/inspections"
	"github.com/raft-tech/konfirm-inspections/internal/logging"
	"github.com/raft-tech/konfirm-inspections/pkg/http"
	"github.com/raft-tech/konfirm-inspections/pkg/storage/source"
)

var (
	logger *zap.Logger

	server        string
	replayEntries []TableEntry

	// Ping Metrics
	pingSuccess  prometheus.Gauge
	pingDuration prometheus.Gauge

	// Replay Metrics
	replaySuccess  *prometheus.GaugeVec
	replayDuration *prometheus.GaugeVec

	labelFilter  ginkgo.LabelFilter
	pingLabels   Labels = []string{"ping"}
	replayLabels Labels = []string{"replay"}
)

func init() {
	inspections.RegisterTestFlags(flag.CommandLine)
}

func TestHTTP(t *testing.T) {

	logger = logging.NewLogger(GinkgoWriter)
	ctx, done := context.WithCancel(logging.NewContext(context.Background(), logger.Named("healthz")))
	defer done()
	inspections.StartHealthz(ctx)

	RegisterTestingT(t)
	RegisterFailHandler(Fail)

	suiteCfg, reporterCfg := GinkgoConfiguration()
	labelFilter = ginkgo.MustParseLabelFilter(suiteCfg.LabelFilter)

	g := NewGomegaWithT(t)

	// Server is the first arg and *must* be defined
	server = flag.CommandLine.Arg(0)
	g.Expect(server).NotTo(BeEmpty(), "a valid server URL is the first argument")

	// If replays are tested, at least one spec arg *must* be defined
	if labelFilter(replayLabels) {
		args := flag.CommandLine.Args()
		g.Expect(len(args)).To(BeNumerically(">=", 2), "at least one spec is defined as the second argument")
		for _, s := range args[1:] {
			spec, err := source.NewSpec(s, "")
			g.Expect(err).NotTo(HaveOccurred(), "validate replay spec")
			replayEntries = append(replayEntries, Entry(spec.Describe(), spec.Describe(), spec.Generate(), spec.Size()))
		}
	}

	setupMetrics()
	RunSpecs(t, "HTTP", suiteCfg, reporterCfg)
}

var _ = Describe("Check", func() {

	It("can ping the server", func(ctx context.Context) {
		ctx = logging.NewContext(ctx, logger)
		client := http.NewClient(server, gohttp.DefaultClient)
		start := time.Now()
		ok, err := client.Check(ctx)
		pingDuration.Set(float64(time.Now().Sub(start).Milliseconds()))
		if ok {
			pingSuccess.Set(1.0)
		} else {
			pingSuccess.Set(0.0)
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
	})

}, pingLabels)

var _ = Describe("ReplayN", func() {

	DescribeTable("replays N bytes", func(ctx context.Context, spec string, src io.Reader, size int64) {
		ctx = logging.NewContext(ctx, logger)
		labels := prometheus.Labels{"spec": spec}
		client := http.NewClient(server, gohttp.DefaultClient)
		start := time.Now()
		ok, err := client.ReplayN(ctx, src, size)
		replayDuration.With(labels).Set(float64(time.Now().Sub(start).Milliseconds()))
		if ok {
			replaySuccess.With(labels).Set(1.0)
		} else {
			replaySuccess.With(labels).Set(0.0)
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(ok).To(BeTrue())
	}, replayEntries)

}, replayLabels)

func setupMetrics() {

	namespace := inspections.MetricsNamespace
	subsystem := "http"
	sharedLabels := prometheus.Labels{
		"server": server,
	}

	pingSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Subsystem:   subsystem,
		Name:        "ping_successful",
		ConstLabels: sharedLabels,
	})

	pingDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Subsystem:   subsystem,
		Name:        "ping_duration_ms",
		ConstLabels: sharedLabels,
	})

	replaySuccess = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   namespace,
		Subsystem:   subsystem,
		Name:        "replay_successful",
		ConstLabels: sharedLabels,
	}, []string{"spec"})

	replayDuration = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   namespace,
		Subsystem:   subsystem,
		Name:        "replay_duration_ms",
		ConstLabels: sharedLabels,
	}, []string{"spec"})
}

var _ = AfterSuite(func(ctx context.Context) {

	metrics := inspections.NewMetrics()

	// Register Ping metrics only if the ping node ran
	if labelFilter(pingLabels) {
		metrics.Register(pingSuccess)
		metrics.Register(pingDuration)
	}

	// Register Replay metrics only if the replay node ran
	if labelFilter(replayLabels) {
		metrics.Register(replaySuccess)
		metrics.Register(replayDuration)
	}

	metrics.Push(ctx)
})
