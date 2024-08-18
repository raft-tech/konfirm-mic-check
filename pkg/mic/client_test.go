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

package mic

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/raft-tech/konfirm-inspections/internal/logging"
)

var server *http.Server

var _ = Describe("Client", func() {

	var mic Client

	It("Checks out", func(ctx context.Context) {
		ctx = logging.NewContext(ctx, logger)
		Expect(mic.Check(ctx)).To(BeTrue())
	})

	It("Replays small tracks", func(ctx context.Context) {
		ctx = logging.NewContext(ctx, logger)
		buf := bytes.NewBuffer([]byte("All work and no play makes Jack a dull boy. 😎"))
		Expect(mic.ReplayN(ctx, buf, int64(buf.Len()))).To(BeTrue())
	})

	BeforeEach(func() {
		mic = NewClient(fmt.Sprintf("http://%s", server.Addr), http.DefaultClient)
	})

})

var _ = BeforeSuite(func() {

	listener, err := net.Listen("tcp", "localhost:0")
	Expect(err).NotTo(HaveOccurred())

	server = &http.Server{
		Addr:    listener.Addr().String(),
		Handler: NewHandler(),
	}
	DeferCleanup(func(ctx context.Context) {
		Expect(server).NotTo(BeNil())
		Expect(server.Shutdown(ctx)).To(Succeed())
	})

	go func() {
		GinkgoRecover()
		Expect(server.Serve(listener)).To(MatchError(http.ErrServerClosed))
	}()
})
