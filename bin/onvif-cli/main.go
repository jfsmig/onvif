// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/jfsmig/onvif/networking"
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
	params := networking.ClientParams{
		Xaddr:      "",
		Username:   envOrDefault("ONVIF_USERNAME", "admin"),
		Password:   envOrDefault("ONVIF_PASSWORD", "admin"),
		HttpClient: nil,
	}

	cmd := &cobra.Command{
		Use:   "main",
		Short: "OnVif command Line Interface",
		Long:  "CLI Client for OnVif devices",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrMissingSubcommand
		},
	}

	cmdDiscover := &cobra.Command{
		Use:   "discover",
		Short: "Discover the local cameras",
		Long:  "Discover the local cameras on all the local network interfaces using the built-in Web Service discovery mechanism",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
			defer cancel()
			return discover(ctx)
		},
	}

	cmdDump := &cobra.Command{
		Use:   "dump",
		Short: "Dump the configuration of the given camera",
		Long:  "Dump the configuration of the given camera, playing all the possible OnVif calls to explicitly check which are supported",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrMissingSubcommand
		},
	}

	cmdDumpAll := &cobra.Command{
		Use:   "all",
		Short: "Dump the configuration of the given camera",
		Long:  "Dump the configuration of the given camera, playing all the possible OnVif calls to explicitly check which are supported",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
			defer cancel()
			params.Xaddr = args[0]
			return dumpAll(ctx, params)
		},
	}

	cmdDumpDescr := &cobra.Command{
		Use:   "descr",
		Short: "Dump a general descriptor of the given camera",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
			defer cancel()
			params.Xaddr = args[0]
			return dumpDescriptor(ctx, params)
		},
	}

	cmdDumpMedia := &cobra.Command{
		Use:   "media",
		Short: "Dump a general descriptor of the given camera",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
			defer cancel()
			params.Xaddr = args[0]
			return dumpMedia(ctx, params)
		},
	}

	cmdDump.AddCommand(cmdDumpDescr, cmdDumpMedia, cmdDumpAll)
	cmd.AddCommand(cmdDiscover, cmdDump)

	if err := cmd.Execute(); err != nil {
		Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		Logger.Info().Msg("Exiting")
	}
}

func runAll(ctx context.Context, hooks ...func(ctx2 context.Context)) {
	for _, fun := range hooks {
		if ctx.Err() != nil {
			return
		}
		fun(ctx)
	}
}

type Runner struct {
	wg sync.WaitGroup
}

func (r *Runner) Async(fn func()) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		fn()
	}()
}

func (r *Runner) Wait() { r.wg.Wait() }

func envOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	} else {
		return defaultValue
	}

}
