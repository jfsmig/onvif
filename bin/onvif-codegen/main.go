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
	"errors"
	"fmt"
	"go/token"
	"os"
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
		Use:   "onvif-codegen",
		Short: "Generate the ONVIF SOAP call wrappers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrMissingSubcommand
		},
	}

	sdk := &cobra.Command{
		Use:   "sdk PACKAGE CALLS_FILE",
		Short: "Generate one Call_<Method> wrapper per method listed in CALLS_FILE",
		Long: "Generate one <Method>_auto.go per method listed in CALLS_FILE, in the\n" +
			"directory holding CALLS_FILE. That directory must be named PACKAGE.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return codegenSdk(args[0], args[1])
		},
	}

	cmd.AddCommand(sdk)

	if err := cmd.Execute(); err != nil {
		Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		Logger.Info().Msg("Exiting")
	}
}

type Method struct {
	Name string
}

// getMethods reads the list of ONVIF method names from sourceFile, one per line, with '#'
// introducing a comment line and anything after the first blank on a line ignored.
//
// Every failure is reported rather than logged and skipped. Silence here used to be
// indistinguishable from success: the scanner error was dropped, so an unreadable source
// yielded an empty method list, the generator wrote no file at all, and `go generate`
// exited 0. Pointing the tool at a directory did exactly that.
func getMethods(sourceFile string) ([]Method, error) {
	fin, err := os.Open(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", sourceFile, err)
	}
	defer func() { _ = fin.Close() }()

	out := make([]Method, 0)
	scanner := bufio.NewScanner(fin)
	scanner.Split(bufio.ScanLines)
	for lineno := 1; scanner.Scan(); lineno++ {
		line := strings.TrimSpace(scanner.Text())

		// The second token is deliberately ignored, not absent: it lets a line carry a
		// trailing comment, and it is where a per-method attribute would go.
		method := strings.SplitN(line, " ", 2)[0]
		if method == "" || strings.HasPrefix(method, "#") {
			continue
		}

		// The name becomes both a Go type reference and a file name, so an entry that is
		// not an identifier would produce either an uncompilable file or, with a '/' or
		// "..", a write outside the target directory.
		if !token.IsIdentifier(method) || !token.IsExported(method) {
			return nil, fmt.Errorf("%s:%d: %q is not an exported Go identifier", sourceFile, lineno, method)
		}

		out = append(out, Method{Name: method})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", sourceFile, err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: no method listed", sourceFile)
	}

	return out, nil
}
