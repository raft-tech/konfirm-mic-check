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
	"github.com/spf13/cobra"
)

func New() *cobra.Command {

	cmd := &cobra.Command{
		Short:         "Verify HTTP connectivity",
		SilenceErrors: true,
		SilenceUsage:  true,
		Use:           "http [COMMAND]",
	}

	server := &cobra.Command{
		RunE:  serve,
		Short: "starts the HTTP server",
		Use:   "serve [--addr ADDRESS]",
	}
	server.PersistentFlags().StringVarP(&serverAddr, "addr", "l", ":8080", "the address the server will listen on")
	server.PersistentFlags().StringVarP(&maxReplayRequest, "max-replay", "m", "128Mi", "the maximum replay request size")

	ping := &cobra.Command{
		RunE:  client,
		Short: "sends a simple GET request to the server at the specified URL",
		Use:   "ping URL",
	}

	replay := &cobra.Command{
		RunE:  client,
		Short: "streams the specified number of bytes to the server at the specified URL",
		Long: "Replay sends the specified number of bytes to the server at the specified URL and expects to receive " +
			"the exact same bytes back. SHA256 digests are calculated for the sent and received bytes, and the two " +
			"are compared. The command is successful only if the HTTP request/response had no error and the digests " +
			"match.",
		Use: "replay URL SPEC [SPEC]...",
	}

	cmd.AddCommand(server, ping, replay)
	return cmd
}
