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

package logging

import (
	"context"

	"go.uber.org/zap"
)

type key string

const loggerKey key = "logger"

var nop = zap.NewNop()

func NewContext(parent context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(parent, loggerKey, logger)
}

func FromContext(ctx context.Context) *zap.Logger {
	logger := nop
	if l, ok := ctx.Value(loggerKey).(*zap.Logger); ok && l != nil {
		logger = l
	}
	return logger
}
