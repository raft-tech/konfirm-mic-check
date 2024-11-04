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

package storage

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/raft-tech/konfirm-inspections/inspections"
	"github.com/raft-tech/konfirm-inspections/internal/cli"
	"github.com/raft-tech/konfirm-inspections/internal/healthz"
	"github.com/raft-tech/konfirm-inspections/internal/logging"
)

var (
	baseDir      string
	maxInstances int
	scrub        bool
)

func New() *cobra.Command {

	cmd := &cobra.Command{
		Use:     "storage [TEST_SPECS]",
		Short:   "Inspect block storage by performing write and read operations",
		Example: "storage small:1KiB medium:128MiB large:2GiB",
		RunE:    storage,
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&baseDir, "base-dir", "", "(required) sets the data directory for inspection data")
	flags.IntVar(&maxInstances, "max-instances", 3, "sets the maximum number of data instances to retain")
	flags.BoolVar(&scrub, "scrub", false, "scrub unindexed files from the volume")

	return cmd
}

func storage(cmd *cobra.Command, cargs []string) error {

	logger := logging.NewLogger(cmd.OutOrStdout())

	// Start health probes if set
	if addr, _ := cmd.Flags().GetString(healthz.ListenFlag); addr != "" {
		if probes, err := healthz.ListenAndServe(cmd.Context(), addr, func(err error) {
			logger.Error("error serving http probes", zap.Error(err))
		}); err == nil {
			probes.Ready(true)
		} else {
			logger.Error("error starting http probe server", zap.Error(err))
		}
	}

	args := inspections.BuildInspectorArgs(cmd.Flags())
	args = append(args,
		"--konfirm.base-dir", baseDir,
		"--konfirm.max-instances", fmt.Sprintf("%d", maxInstances),
	)

	if scrub {
		args = append(args, "--konfirm.scrub")
	}

	var inspection *exec.Cmd
	if i, e := exec.LookPath("konfirm-storage"); e == nil {
		inspection = exec.CommandContext(cmd.Context(), i, append(args, cargs...)...)
	} else {
		return cli.ErrorF(1, "storage inspector not found")
	}

	var pout io.ReadCloser
	if r, err := inspection.StdoutPipe(); err == nil {
		pout = r
	} else {
		return cli.Wrap(1, err)
	}

	var perr io.ReadCloser
	if r, err := inspection.StderrPipe(); err == nil {
		perr = r
	} else {
		return cli.Wrap(1, err)
	}

	if e := inspection.Start(); e != nil {
		return cli.Wrap(1, e)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(os.Stdout, pout)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(os.Stderr, perr)
	}()
	wg.Wait()

	if e := inspection.Wait(); e == nil {
		return e
	} else if err, ok := e.(*exec.ExitError); ok {
		return cli.Wrap(err.ExitCode(), e)
	} else {
		return cli.Wrap(1, e)
	}
}
