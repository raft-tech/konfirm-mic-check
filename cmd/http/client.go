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
	"os/exec"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/raft-tech/konfirm-inspections/inspections"
	"github.com/raft-tech/konfirm-inspections/internal/cli"
	"github.com/raft-tech/konfirm-inspections/internal/healthz"
	"github.com/raft-tech/konfirm-inspections/internal/logging"
)

var (
	metricsGateway string
	replayBytes    int
)

func client(cmd *cobra.Command, cargs []string) error {

	logger := logging.NewLogger(cmd.OutOrStdout())

	// Start healthz server if set
	if addr, _ := cmd.Flags().GetString(healthz.ListenFlag); addr != "" {
		if probes, err := healthz.ListenAndServe(cmd.Context(), addr, func(err error) {
			logger.Error("error serving http probes", zap.Error(err))
		}); err == nil {
			probes.Ready(true)
		} else {
			logger.Error("error starting http probe server", zap.Error(err))
		}
	}

	// Build the inspection args
	args := inspections.BuildInspectorArgs(cmd.Flags())
	args = append(args, "--ginkgo.label-filter="+cmd.Name())

	// Execute the inspection
	var inspection *exec.Cmd
	if i, e := exec.LookPath("konfirm-http"); e == nil {
		inspection = exec.CommandContext(cmd.Context(), i, append(args, cargs...)...)
	} else {
		return cli.ErrorF(1, "http inspection not found")
	}
	logger.Info("starting http inspection with " + cmd.Name())
	return inspections.Run(inspection, cmd)
}
