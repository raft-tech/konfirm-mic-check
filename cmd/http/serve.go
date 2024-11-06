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
	"errors"
	gohttp "net/http"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/raft-tech/konfirm-inspections/internal/cli"
	"github.com/raft-tech/konfirm-inspections/internal/healthz"
	"github.com/raft-tech/konfirm-inspections/internal/logging"
	"github.com/raft-tech/konfirm-inspections/pkg/http"
)

var (
	serverAddr       string
	maxReplayRequest string
)

func serve(cmd *cobra.Command, _ []string) (err error) {

	logger := logging.NewLogger(cmd.OutOrStdout())

	// Start healthz server if set
	ready := func(_ bool) {}
	if addr, _ := cmd.Flags().GetString(healthz.ListenFlag); addr != "" {
		logger.Debug("starting healthz server", zap.String("address", addr))
		if probes, err := healthz.ListenAndServe(cmd.Context(), addr, func(err error) {
			logger.Error("error serving http probes", zap.Error(err))
		}); err == nil {
			ready = probes.Ready
			logger.Info("healthz started", zap.String("address", addr))
		} else {
			logger.Error("error starting http probe server", zap.Error(err))
		}
	}

	// Configure and the server
	http.SetServerLogger(logger)

	if qty, err := resource.ParseQuantity(maxReplayRequest); err != nil {
		return cli.Wrap(2, errors.Join(errors.New("error parsing max-replay"), err))
	} else if i, ok := qty.AsInt64(); ok {
		if i > 536870912 { // If greater than 512Mi
			logger.Warn("max-replay is set to a high number; large replay requests may result in out-of-memory errors")
		}
		http.MaxReplayRequestSize = i
	} else {
		return cli.ErrorF(2, "max-replay value is too large")
	}

	server := gohttp.Server{
		Addr:    serverAddr,
		Handler: http.NewHandler(),
	}

	// Start
	done := make(chan error)
	go func(out chan<- error) {
		logger.Info("starting server", zap.String("address", serverAddr))
		ready(true)
		out <- server.ListenAndServe()
		close(out)
	}(done)

	select {

	// Normal shutdowns
	case <-cmd.Context().Done():

		// Perform shutdown
		logger.Info("initiating shutdown")
		ready(false)
		ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Second, errors.New("server shutdown timed out"))
		defer cancel()
		if err = server.Shutdown(ctx); err == nil {
			logger.Info("shutdown complete")
		} else {
			logger.Error("an error occurred while shutting down the server", zap.Error(err))
		}

		// Drain the done channel
		for e := range done {
			if !errors.Is(e, gohttp.ErrServerClosed) {
				logger.Error("a server error occurred", zap.Error(e))
				if err == nil {
					err = e
				}
			}
		}

	// Server cli
	case e := <-done:
		if !errors.Is(e, gohttp.ErrServerClosed) {
			logger.Error("a server error occurred", zap.Error(e))
			err = e
		}
	}

	_ = logger.Sync()
	return
}
