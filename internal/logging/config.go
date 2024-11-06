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
	"flag"
	"io"
	"strings"

	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	LogLevelFlag      = "log-level"
	LogFormatFlag     = "log-format"
	TestLogLevelFlag  = "konfirm.log-level"
	TestLogFormatFlag = "konfirm.log-format"
)

var (
	logLevel  = "INFO"
	logFormat = "console"
)

func RegisterCmdFlags(set *pflag.FlagSet) {
	set.StringVar(&logLevel, LogLevelFlag, logLevel, "sets the log level to one of: DEBUG, INFO, WARN, ERROR")
	set.StringVar(&logFormat, LogFormatFlag, logFormat, "sets the log format to one of: console, json")
}

func RegisterTestFlags(set *flag.FlagSet) {
	set.StringVar(&logLevel, TestLogLevelFlag, logLevel, "sets the log level to one of: DEBUG, INFO, WARN, ERROR")
	set.StringVar(&logFormat, TestLogFormatFlag, logFormat, "sets the log format to one of: console, json")
}

func NewLogger(out io.Writer) *zap.Logger {

	logLevel = strings.TrimSpace(strings.ToUpper(logLevel))
	logFormat = strings.TrimSpace(strings.ToLower(logFormat))

	// Configure the log encoder
	lecfg := zap.NewProductionEncoderConfig()
	lenc := zapcore.NewJSONEncoder(lecfg)
	if logFormat == "console" {
		lecfg.EncodeTime = zapcore.ISO8601TimeEncoder
		lenc = zapcore.NewConsoleEncoder(lecfg)
	}

	// Configure the log level
	var llevel zapcore.Level
	switch logLevel {
	case "DEBUG":
		llevel = zapcore.DebugLevel
	case "INFO":
		llevel = zapcore.InfoLevel
	case "WARN":
		llevel = zapcore.WarnLevel
	case "ERROR":
		llevel = zapcore.ErrorLevel
	}

	return zap.New(zapcore.NewCore(lenc, zapcore.AddSync(out), zapcore.LevelOf(llevel)))
}
