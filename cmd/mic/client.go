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
	"net/http"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/raft-tech/konfirm-inspections/internal/cli"
	"github.com/raft-tech/konfirm-inspections/internal/logging"
	"github.com/raft-tech/konfirm-inspections/pkg/mic"
)

func client(cmd *cobra.Command, args []string) error {

	if len(args) < 1 {
		return cli.ErrorF(2, "a server URL is required as the first argument")
	}

	var local mic.Client
	if addr, err := url.Parse(args[0]); err != nil {
		return cli.ErrorF(2, "malformed server URL: %s", args[0])
	} else if addr.Scheme != "http" && addr.Scheme != "https" {
		return cli.ErrorF(2, "scheme must be either http or https")
	} else if addr.Hostname() == "" {
		return cli.ErrorF(2, "a host is required")
	} else if addr.RawQuery != "" {
		return cli.ErrorF(2, "queries are not supported in the base URL")
	} else if addr.RawFragment != "" {
		return cli.ErrorF(2, "fragments are not supported in the base URL")
	} else {
		local = mic.NewClient(addr.String(), http.DefaultClient)
	}

	ctx := logging.NewContext(cmd.Context(), logging.NewLogger(cmd.OutOrStdout()))

	switch cmd.Name() {
	case "check":
		if ok, err := local.Check(ctx); err != nil {
			return cli.ErrorF(1, "an error occurred while running check")
		} else if !ok {
			return cli.ErrorF(1, "check failed")
		} else {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Check successful!")
		}
	}

	return nil
}
