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
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/raft-tech/konfirm-inspections/internal/logging"
	"github.com/raft-tech/konfirm-inspections/pkg/mic"
)

func serve(cmd *cobra.Command, _ []string) (err error) {

	logger := logging.NewLogger(cmd.OutOrStdout())
	mic.SetServerLogger(logger)

	server := http.Server{
		Addr:    serverAddr,
		Handler: mic.NewHandler(),
	}

	done := make(chan error)
	go func(out chan<- error) {
		logger.Info("starting server", zap.String("address", serverAddr))
		out <- server.ListenAndServe()
		close(out)
	}(done)

	select {

	// Normal shutdowns
	case <-cmd.Context().Done():

		// Perform shutdown
		logger.Info("initiating shutdown")
		ctx, cancel := context.WithTimeoutCause(context.Background(), 10*time.Second, errors.New("server shutdown timed out"))
		defer cancel()
		if err = server.Shutdown(ctx); err == nil {
			logger.Info("shutdown complete")
		} else {
			logger.Error("an error occurred while shutting down the server", zap.Error(err))
		}

		// Drain the done channel
		for e := range done {
			if !errors.Is(e, http.ErrServerClosed) {
				logger.Error("a server error occurred", zap.Error(e))
				if err == nil {
					err = e
				}
			}
		}

	// Server cli
	case e := <-done:
		if !errors.Is(e, http.ErrServerClosed) {
			logger.Error("a server error occurred", zap.Error(e))
			err = e
		}
	}

	_ = logger.Sync()
	return
}
