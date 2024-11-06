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

package inspections

import (
	"github.com/spf13/pflag"

	"github.com/raft-tech/konfirm-inspections/internal/logging"
)

func BuildInspectorArgs(f *pflag.FlagSet) (args []string) {

	if str, err := f.GetString(MetricsGatewayFlag); err == nil {
		args = append(args, "--"+TestMetricsGatewayFlag, str)
	}

	if str, err := f.GetString(MetricsJobFlag); err == nil {
		args = append(args, "--"+TestMetricsJobFlag, str)
	}

	if str, err := f.GetString(MetricsInstanceFlag); err == nil {
		args = append(args, "--"+TestMetricsInstanceFlag, str)
	}

	if str, err := f.GetString(logging.LogLevelFlag); err == nil {
		args = append(args, "--"+logging.TestLogLevelFlag, str)
	}

	if str, err := f.GetString(logging.LogFormatFlag); err == nil {
		args = append(args, "--"+logging.TestLogFormatFlag, str)
	}

	return
}
