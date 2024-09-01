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
	"github.com/spf13/cobra"
)

var serverAddr string

func New() *cobra.Command {

	mic := &cobra.Command{
		Short:         "Verify network connectivity",
		SilenceErrors: true,
		SilenceUsage:  true,
		Use:           "mic COMMAND",
	}

	server := &cobra.Command{
		RunE:  serve,
		Short: "starts the HTTP server",
		Use:   "serve [--addr ADDRESS]",
	}
	server.PersistentFlags().StringVarP(&serverAddr, "addr", "l", ":8080", "the address the server will listen on")

	check := &cobra.Command{
		RunE:  client,
		Short: "checks the server at the specified URL",
		Use:   "check URL",
	}

	mic.AddCommand(server, check)
	return mic
}
