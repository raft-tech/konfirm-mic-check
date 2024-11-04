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
	"errors"
	"io"
	"os/exec"
	"sync"

	"github.com/spf13/cobra"

	"github.com/raft-tech/konfirm-inspections/internal/cli"
)

func Run(inspection *exec.Cmd, parent *cobra.Command) error {

	var iout io.ReadCloser
	if r, err := inspection.StdoutPipe(); err == nil {
		iout = r
	} else {
		return cli.Wrap(1, err)
	}

	var ierr io.ReadCloser
	if r, err := inspection.StderrPipe(); err == nil {
		ierr = r
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
		_, _ = io.Copy(parent.OutOrStdout(), iout)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(parent.OutOrStderr(), ierr)
	}()
	wg.Wait()

	err := &exec.ExitError{}
	if e := inspection.Wait(); e == nil {
		return e
	} else if errors.As(e, &err) {
		return err
	} else {
		return cli.Wrap(1, e)
	}
}
