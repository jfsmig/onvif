// Copyright (C) 2022-2026 Jean-Francois SMIGIELSKI
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	// Logger is a zerolog logger, that can be safely used from any part of the application.
	// It gathers the format and the output. The application can replace the default Logger
	// for an alternative that meets its own output.
	Logger = zerolog.
		New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().Timestamp().
		Logger()
)

var (
	ErrMissingSubcommand = errors.New("missing sub-command")
)

func main() {
	cmd := &cobra.Command{
		Use:   "codegen",
		Short: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrMissingSubcommand
		},
	}

	sdk := &cobra.Command{
		Use:   "sdk",
		Short: "Generate the files of a SDK package",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
			defer cancel()
			return codegenSdk(ctx, args[0], args[1])
		},
	}

	cmd.AddCommand(sdk)

	if err := cmd.Execute(); err != nil {
		Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		Logger.Info().Msg("Exiting")
	}
}

func getwd() string {
	path, _ := os.Getwd()
	return path
}

type Method struct {
	Name string
	// TODO(jfsmig): optional fields?
}

func getMethods(sourceFile string) []Method {
	out := make([]Method, 0)

	fin, err := os.Open(sourceFile)
	if err != nil {
		Logger.Fatal().Err(err).Str("wd", getwd()).Str("file", sourceFile).Msg("Failed to open the configuration file")
	}
	defer func() { _ = fin.Close() }()

	scanner := bufio.NewScanner(fin)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		tokens := strings.SplitN(line, " ", 2)
		method := tokens[0]
		if method == "" || strings.HasPrefix(method, "#") {
			continue
		}

		out = append(out, Method{Name: method})
	}

	return out
}
