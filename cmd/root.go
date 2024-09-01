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

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/raft-tech/konfirm-inspections/cmd/mic"
	"github.com/raft-tech/konfirm-inspections/cmd/storage"
	"github.com/raft-tech/konfirm-inspections/internal/logging"
)

func New() *cobra.Command {
	root := &cobra.Command{
		Use:           "",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	logging.RegisterPFlags(root.PersistentFlags())
	root.AddCommand(mic.New(), storage.New())
	return root
}
