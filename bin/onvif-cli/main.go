// Copyright (c) 2022-2022 Jean-Francois SMIGIELSKI

package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
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
	httpClient = http.Client{}

	auth = networking.ClientAuth{
		Username: envOrDefault("ONVIF_USERNAME", "admin"),
		Password: envOrDefault("ONVIF_PASSWORD", "admin"),
	}
)

var (
	ErrMissingSubcommand = errors.New("missing sub-command")
)

func main() {

	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, time.Minute)
	defer cancel()

	cmd := &cobra.Command{
		Use:   "main",
		Short: "OnVif command Line Interface",
		Long:  "CLI Client for OnVif devices",
		RunE:  func(cmd *cobra.Command, args []string) error { return ErrMissingSubcommand },
	}

	cmdDiscover := &cobra.Command{
		Use:     "discover",
		Aliases: []string{"find", "crawl", "probe"},
		Short:   "Discover the local cameras",
		Args:    cobra.NoArgs,
		RunE:    func(cmd *cobra.Command, args []string) error { return discover(ctx, false) },
	}

	cmdStreams := &cobra.Command{
		Use:     "streams",
		Aliases: []string{"streams", "stream"},
		Short:   "Print the stream URL for the cameras locally discovered",
		Args:    cobra.NoArgs,
		RunE:    func(cmd *cobra.Command, args []string) error { return discover(ctx, true) },
	}

	cmdDump := &cobra.Command{
		Use:     "dump",
		Aliases: []string{"all", "detail", "details"},
		Short:   "Dump the configuration of the given camera",
		Args:    cobra.NoArgs,
		RunE:    func(cmd *cobra.Command, args []string) error { return ErrMissingSubcommand },
	}

	cmdDumpAll := &cobra.Command{
		Use:     "all",
		Aliases: []string{"full"},
		Short:   "Dump the configuration of the given camera",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dumpAll(ctx, networking.ClientInfo{Xaddr: args[0]})
		},
	}

	cmdDumpDescr := &cobra.Command{
		Use:     "descriptor",
		Aliases: []string{"minimal", "mini"},
		Short:   "Dump a general descriptor of the given camera",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dumpDescriptor(ctx, networking.ClientInfo{Xaddr: args[0]})
		},
	}

	cmdDumpMedia := &cobra.Command{
		Use:   "media",
		Short: "Dump the information related to the Media service of the camera",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dumpMedia(ctx, networking.ClientInfo{Xaddr: args[0]})
		},
	}

	cmdDumpPtz := &cobra.Command{
		Use:     "ptz",
		Aliases: []string{"PTZ"},
		Short:   "Dump the information related to the PTZ service of the camera",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dumpPTZ(ctx, networking.ClientInfo{Xaddr: args[0]})
		},
	}

	cmdDumpEvents := &cobra.Command{
		Use:     "event",
		Aliases: []string{"events", "evt"},
		Short:   "Dump the information related to the Events service of the camera",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dumpEvents(ctx, networking.ClientInfo{Xaddr: args[0]})
		},
	}

	cmdDumpProfiles := &cobra.Command{
		Use:     "profile",
		Aliases: []string{"profiles", "prof"},
		Short:   "Dump the information related to the Profiles of the camera",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dumpProfiles(ctx, networking.ClientInfo{Xaddr: args[0]})
		},
	}

	cmdDumpDevice := &cobra.Command{
		Use:     "device",
		Aliases: []string{"devices", "dev"},
		Short:   "Dump the information related to the core Device service of the camera",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return dumpDevice(ctx, networking.ClientInfo{Xaddr: args[0]})
		},
	}

	cmdDump.AddCommand(cmdDumpDescr, cmdDumpAll)
	cmdDump.AddCommand(cmdDumpMedia, cmdDumpPtz, cmdDumpEvents, cmdDumpProfiles, cmdDumpDevice)
	cmd.AddCommand(cmdDiscover, cmdStreams, cmdDump)

	if err := cmd.Execute(); err != nil {
		Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		Logger.Info().Msg("Exiting")
	}
}

func envOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	} else {
		return defaultValue
	}
}
