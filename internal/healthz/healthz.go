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

package healthz

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	ListenFlag = "healthz"
)

func LivenessHandler() http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, _ *http.Request) {
		res.Header().Add("Content-Type", "text/plain")
		_, _ = res.Write([]byte("OK"))
	})
}

func ReadinessHandler() (http.Handler, func(bool)) {
	var ready bool
	isReady := func(ok bool) {
		ready = ok
	}
	return http.HandlerFunc(func(res http.ResponseWriter, _ *http.Request) {
		res.Header().Add("Content-Type", "text/plain")
		if ready {
			_, _ = res.Write([]byte("READY"))
		} else {
			res.WriteHeader(http.StatusServiceUnavailable)
			_, _ = res.Write([]byte("NOT READY"))
		}

	}), isReady
}

type Server interface {

	// Address returns the address the server is listening on
	Address() string

	// Ready sets to the server's readiness to the specified value
	Ready(bool)
}

// ListenAndServe starts a new metric server listening at the specified address.
// The server will continue listening until the provided context is canceled or an error occurs.
// Any error returned by the underlying [http.Serve] func will be passed to the provided onError
// function (if it is not nil).
func ListenAndServe(ctx context.Context, addr string, onError func(err error)) (Server, error) {

	if onError == nil {
		onError = func(_ error) {}
	}

	var listener net.Listener
	if l, err := net.Listen("tcp", addr); err == nil {
		listener = l
	} else {
		return nil, err
	}

	srv := &server{
		addr: listener.Addr().String(),
		http: http.NewServeMux(),
	}

	srv.http.Handle("/", LivenessHandler())

	var readiness http.Handler
	readiness, srv.isReady = ReadinessHandler()
	srv.http.Handle("/ready", readiness)
	srv.http.Handle("/metrics", promhttp.Handler())

	go func(ctx context.Context, l net.Listener, srv *server, onError func(err error)) {

		// Create an HTTP Server
		hsrv := http.Server{
			BaseContext: func(_ net.Listener) context.Context {
				return ctx
			},
			Handler: srv.http,
		}

		// Serve on the provided listener
		err := make(chan error)
		go func() {
			defer close(err)
			if e := hsrv.Serve(l); !errors.Is(e, http.ErrServerClosed) {
				err <- e
			}
		}()

		// Wait for server error or context closure
		select {
		case _ = <-ctx.Done():
			sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if e := hsrv.Shutdown(sctx); e != nil {
				onError(e)
			}
			for e := range err {
				if e != nil {
					onError(e)
				}
			}
		case e := <-err:
			onError(e)
		}
	}(ctx, listener, srv, onError)

	return srv, nil
}

type server struct {
	addr    string
	isReady func(ok bool)
	http    *http.ServeMux
}

func (s *server) Address() string {
	return s.addr
}

func (s *server) Ready(ok bool) {
	s.isReady(ok)
}
