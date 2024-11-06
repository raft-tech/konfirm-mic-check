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
	"bytes"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
)

const (
	micCheck      = "Mic check. One two. One two."
	contentType   = "Content-Type"
	contentLength = "Content-Length"
)

var (
	MaxReplayRequestSize int64 = 128 * 1024 * 1024 // 128 MiB
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
		logger.Info("unsupported request", zap.String("method", req.Method))
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Request size must not exceed MaxReplayRequestSize
	if m := MaxReplayRequestSize; req.ContentLength > m {
		logger.Warn("request size exceeds maximum", zap.Int64("size", req.ContentLength), zap.Int64("maxSize", m))
		res.WriteHeader(http.StatusRequestEntityTooLarge)
		return
	}

	// Write request body into buffer
	buf := &bytes.Buffer{}
	if n, err := buf.ReadFrom(io.LimitReader(req.Body, req.ContentLength)); err != nil {
		logger.Error("error reading request body", zap.Error(err))
		res.WriteHeader(http.StatusInternalServerError)
		return
	} else if n != req.ContentLength {
		logger.Error("request content did not match expected size", zap.Int64("size", n), zap.Int64("expected", req.ContentLength))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Set Content-Type and Content-Length response headers
	headers := res.Header()
	headers.Set(contentLength, fmt.Sprintf("%d", req.ContentLength))
	if ct := req.Header.Get(contentType); ct != "" {
		headers.Set(contentType, ct)
	}

	// Write the response body
	if _, err := io.Copy(res, buf); err != nil {
		logger.Error("error writing response body", zap.Error(err))
	} else {
		logger.Info("response sent successfully")
	}
}
