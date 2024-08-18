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
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

const (
	micCheck      = "Mic mic. One two. One two."
	contentType   = "Content-Type"
	contentLength = "Content-Length"
)

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/check", check)
	mux.HandleFunc("/replay", replay)
	return mux
}

func logRequest(logger *zap.Logger, req *http.Request) {
	logger.Info("new request", zap.String("clientAddr", req.RemoteAddr), zap.String("method", req.Method), zap.String("uri", req.RequestURI))
}

func check(res http.ResponseWriter, req *http.Request) {
	logger := logger.Named("server").With(zap.String("handler", "mic"))
	logRequest(logger, req)

	// Only support GET requests
	if req.Method != http.MethodGet {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	output := []byte(micCheck)

	headers := res.Header()
	headers.Set(contentType, "text/plain")
	headers.Set(contentLength, fmt.Sprintf("%d", len(output)))
	res.WriteHeader(http.StatusOK)

	for i := 0; i < len(output); {
		n, err := res.Write(output[i:])
		if err != nil {
			logger.Error("error handling request", zap.Error(err))
			return
		}
		i += n
	}
}

func replay(res http.ResponseWriter, req *http.Request) {

	logger := logger.Named("server").With(zap.String("handler", "replay"))
	logRequest(logger, req)

	// Only support POST requests
	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Set Content-Type and Content-Length response headers
	headers := res.Header()
	headers.Set(contentLength, fmt.Sprintf("%d", req.ContentLength))
	if ct := req.Header.Get(contentType); ct != "" {
		headers.Set(contentType, ct)
	}

	// Echo the request body
	for i, j := int64(0), req.ContentLength; i < req.ContentLength; {
		n, err := io.CopyN(res, req.Body, j)
		if err != nil {
			logger.Error("error handling request", zap.Error(err))
			return
		}
		i += n
		j -= n // Reduce max read bytes by previously read bytes
	}
}
