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

package http_test

import (
	"context"
	"flag"
	gohttp "net/http"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/raft-tech/konfirm-inspections/cmd/http"
	"github.com/raft-tech/konfirm-inspections/internal/healthz"
)

var (
	serverAddr      string
	serverProbeAddr string
	clientProbeAddr string
)

func init() {
	flag.CommandLine.StringVar(&serverAddr, "konfirm.server-addr", "localhost:8080", "sets the listening address for the server during testing")
	flag.CommandLine.StringVar(&serverProbeAddr, "konfirm.server-probe-addr", "localhost:8081", "sets the listening address for server probes during testing")
	flag.CommandLine.StringVar(&clientProbeAddr, "konfirm.probe-addr", "localhost:8082", "sets the listening address for probes during testing")
}

func TestHttpCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HTTP")
}

var _ = Describe("command", func() {

	Context("with server", func() {

		It("checks", func(ctx context.Context) {
			cmd := http.New()
			cmd.PersistentFlags().String(healthz.ListenFlag, serverProbeAddr, "")
			cmd.SetArgs([]string{"ping", "http://" + serverAddr})
			cmd.SetOut(GinkgoWriter)
			cmd.SetErr(GinkgoWriter)
			Expect(cmd.ExecuteContext(ctx)).To(Succeed())
		})

		BeforeEach(func(ctx context.Context) {

			// Creat the http command and add flags defined in Root
			server := http.New()
			server.PersistentFlags().String(healthz.ListenFlag, serverProbeAddr, "")
			server.SetOut(GinkgoWriter)
			server.SetErr(GinkgoWriter)

			// Run the server subcommand
			sctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
			DeferCleanup(func() {
				cancel()
			})
			go func(ctx context.Context) {
				defer GinkgoRecover()
				server.SetArgs([]string{"serve", "--addr", serverAddr})
				Expect(server.ExecuteContextC(ctx)).To(Succeed())
			}(sctx)

			// Wait for the server to be available
			addr := serverProbeAddr
			if strings.HasPrefix(addr, ":") {
				addr = "localhost" + addr
			}
			Eventually(func() (*gohttp.Response, error) {
				return gohttp.Get("http://" + addr + "/ready")
			}).Should(HaveHTTPStatus(gohttp.StatusOK))
		})
	})

})
