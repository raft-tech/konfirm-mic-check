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
	"crypto"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/raft-tech/konfirm-inspections/internal/logging"
)

var HttpStatusCodeErr = errors.New("the server responded with an unsuccessful HTTP status code")
var ExceedsMaxRequestSizeErr = errors.New("request exceeded the server's maximum permitted size")

type Client interface {
	Check(ctx context.Context) (bool, error)
	ReplayN(ctx context.Context, body io.Reader, len int64) (bool, error)
}

func NewClient(remoteAddr string, httpClient *http.Client) Client {
	if httpClient == nil {
		panic("httpClient must not be nil")
	}
	return &client{
		http:   httpClient,
		server: strings.TrimSuffix(remoteAddr, "/"),
	}
}

type client struct {
	http   *http.Client
	server string
}

func (c *client) Check(ctx context.Context) (bool, error) {

	logger := logging.FromContext(ctx).Named("client").With(zap.String("server", c.server))
	logger.Info("starting check")

	res, err := c.http.Get(fmt.Sprintf("%s/check", c.server))
	if err != nil {
		logger.Error("an error occurred during check", zap.Error(err))
		return false, err
	}

	if res.ContentLength != int64(len(micCheck)) {
		logger.Error("unexpected content-length in check response", zap.Int("expected", len(micCheck)), zap.Int64("actual", res.ContentLength))
		return false, nil
	}

	if body, err := io.ReadAll(res.Body); err != nil {
		logger.Error("an error occurred while reading the check response", zap.Error(err))
		return false, err
	} else if b := string(body); b != micCheck {
		logger.Warn("check response did not matched expected string", zap.String("expected", micCheck), zap.String("actual", b))
		return false, nil
	}

	logger.Info("check successful")
	return true, nil
}

func (c *client) ReplayN(ctx context.Context, body io.Reader, len int64) (bool, error) {

	logger := logging.FromContext(ctx).Named("client").With(zap.String("server", c.server))

	// Tee Body to calculate a digest as it's read/sent
	expected := crypto.SHA256.New()
	body = io.TeeReader(body, expected)

	var req *http.Request
	if r, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/replay", c.server), body); err == nil {
		r.ContentLength = len
		r.Header.Set(contentType, "application/octet-stream")
		req = r
	} else {
		logger.Error("an error occurred generating the replay request", zap.Error(err))
	}

	var res *http.Response
	logger.Debug("initiating replay request", zap.Int64("len", req.ContentLength))
	if r, err := c.http.Do(req); err == nil {
		res = r
		defer func() {
			_ = res.Body.Close()
		}()
		logger.Info("received replay response", zap.Int("code", res.StatusCode), zap.Int64("len", res.ContentLength), zap.String("content", res.Header.Get(contentType)))
	} else {
		logger.Error("replay request failed", zap.Error(err))
	}

	// Validate the response headers
	if res.StatusCode != http.StatusOK {
		err := HttpStatusCodeErr
		if res.StatusCode == http.StatusRequestEntityTooLarge {
			err = ExceedsMaxRequestSizeErr
			logger.Error("replay request failed because it exceed the server's maximum request size")
		} else {
			logger.Error("replay request failed with an non-200 HTTP status code", zap.Int("statusCode", res.StatusCode))
		}
		return false, err
	} else if res.ContentLength != req.ContentLength {
		logger.Error(
			"replay request failed because the response content-length did not match the request length",
			zap.Int64("reqContentLength", req.ContentLength),
			zap.Int64("resContentLength", res.ContentLength),
		)
		return false, nil
	} else if res.Header.Get(contentType) != "application/octet-stream" {
		logger.Error(
			"replay request failed because the response content-type was not 'application/octet-stream",
			zap.String("resContentType", res.Header.Get(contentType)),
		)
		return false, nil
	}

	// Validate the response body
	actual := crypto.SHA256.New()
	if n, err := io.CopyN(actual, res.Body, res.ContentLength); err != nil {
		logger.Error("an error occurred while reading the response", zap.Error(err))
		return false, nil
	} else if n != res.ContentLength {
		logger.Error("response body length did not match specified content length", zap.Int64("actual", n), zap.Int64("expected", res.ContentLength))
		return false, nil
	}

	// Compare digests to determine success
	if exp, act := "sha256:"+hex.EncodeToString(expected.Sum(nil)), "sha256:"+hex.EncodeToString(actual.Sum([]byte(nil))); exp == act {
		logger.Info("replay successful", zap.String("digest", act))
		return true, nil
	} else {
		logger.Warn("response body did not match request body", zap.String("expectedDigest", exp), zap.String("actualDigest", act))
		return false, nil
	}
}
