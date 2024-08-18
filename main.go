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

package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/raft-tech/konfirm-inspections/cmd"
	"github.com/raft-tech/konfirm-inspections/internal/cli"
)

var (
	//go:embed VERSION
	version string
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	app := cmd.New()
	app.Version = version
	if err := app.ExecuteContext(ctx); err != nil {
		code := 1
		var xerr cli.ExitError
		if errors.As(err, &xerr) {
			_, _ = fmt.Fprintln(os.Stderr, xerr.Error())
			code = xerr.ExitCode()
		}
		os.Exit(code)
	}
}
