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
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jfsmig/onvif/credentials"
	"github.com/jfsmig/onvif/networking"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	// Logger gathers the format and the destination of the diagnostics written here, and is
	// a variable so that it can be replaced: an application wanting JSON rather than the
	// console writer, or a file rather than stderr, assigns its own.
	//
	// Ownership, stated the way networking.Client states it for its own fields: reading it is
	// safe from any goroutine -- that is what zerolog is built for -- but replacing it is a
	// plain assignment to a package variable that every fan-out goroutine reads. That belongs
	// in initialisation, before the first call. A swap while calls are in flight is a data
	// race, and no lock here can cover it: the previous wording, "can be safely used from any
	// part of the application", read as permission to do exactly that.
	Logger = zerolog.
		New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().Timestamp().
		Logger()
)

var (
	// Per-request backstop, and for `subscribe` the only budget there is: a one-shot command
	// bounds its whole run with oneShotDeadline, while a subscription is meant to outlive any
	// deadline, so this is what stops one silent camera holding a goroutine for ever. It
	// bounds a single exchange, which for a PullMessages long poll is why pullTimeout has to
	// stay under it.
	//
	// The connection cap is the counterpart of the sdk fan-out. Every Fetch* now issues its
	// independent calls at once, so `dump all` offers dozens of requests simultaneously,
	// and http.Transport defaults to no per-host limit with only two idle connections kept.
	// An embedded camera web server accepts a handful of connections and resets the rest,
	// which would make the concurrent dump slower and flakier than the sequential one it
	// replaced. MaxConnsPerHost queues the surplus inside Do -- still honouring the
	// context -- instead of letting the device refuse it.
	//
	// The policy lives here rather than in the library: sdk takes the caller's *http.Client
	// as given, and a caller with a different device deserves a different number.
	httpClient = http.Client{
		Timeout: networking.DefaultTimeout,
		Transport: &http.Transport{
			MaxConnsPerHost:     4,
			MaxIdleConnsPerHost: 4,
		},
	}

	// resolver answers "which credentials for this camera?" and is built once, by the
	// root command's PersistentPreRunE, before any camera is contacted. It replaces the
	// single ClientAuth this tool used to send to every device of a run.
	//
	// Package-level like httpClient above, and for the same reason: there is exactly one
	// per process, and every command needs it.
	resolver credentials.Resolver

	// baseDir is where --basedir puts its argument. Empty means "not named", which is
	// what tells a mistyped path -- an error -- from a default location nobody created.
	baseDir string

	// verbosity counts the -v flags. Its meaning is verbosityLevel's.
	verbosity int
)

// verbosityLevel turns a count of -v flags into the level below which nothing is printed.
//
// The default is warn, not info, so that a run in which everything worked prints nothing
// at all: the tool's output is what goes to stdout, and an operator piping a dump into jq
// should not have to read a running commentary beside it. Warnings and errors are not
// verbosity -- a credentials file readable by every local account, an interface whose probe
// failed, a camera that answered nothing -- so they are printed whatever the count.
//
// The steps are chosen by what an operator is looking for when they add a v. One says what
// the tool is doing and why it is taking so long; two says what it found on disk and which
// credential it picked, which is the pair of questions a 401 raises; three is the wire,
// where zerolog's trace level already carries the discovery probe and every per-call
// failure that sdk swallows.
//
// Pure over its argument, so the whole policy is one table in a test rather than a run of
// the binary.
func verbosityLevel(count int) zerolog.Level {
	switch count {
	case 0:
		return zerolog.WarnLevel
	case 1:
		return zerolog.InfoLevel
	case 2:
		return zerolog.DebugLevel
	default:
		return zerolog.TraceLevel
	}
}

// oneShotDeadline bounds a command that does a bounded amount of work and then exits.
//
// It used to sit in main(), around the whole process, and that was right while every command
// was one-shot: a discovery probe waits out a fixed collection window and a dump is a few
// dozen round trips, so a run that has not finished within a minute is stuck rather than
// slow. `subscribe` is the case that argument does not cover -- its normal life is measured
// in days, and a cap in main() would have ended it at one minute, silently and successfully
// -- so the budget belongs to each command instead of to the process.
//
// One number for all of them, deliberately: it was chosen for the slowest one-shot there
// is, a `dump all` behind an identifier lookup, and a per-command table would be four
// numbers nobody could justify against each other.
const oneShotDeadline = time.Minute

// runOneShot runs a command, or the bounded phase of one, under oneShotDeadline.
//
// The cancel is deferred here rather than installed once in the root's PersistentPreRunE.
// There the derived context has to outlive the hook, so its cancel could only be dropped --
// `ctx, _ = context.WithTimeout(...)`, which `go vet`'s lostcancel check does reject and CI
// gates on -- or carried in a variable and released from a PersistentPostRun, which cobra
// runs only for the closest hook in the chain and not at all when RunE fails. Here the
// deferred cancel is released by the end of the work it bounds, which is what it means.
//
// A streaming command wraps only the phase that has an end. What it must not do is create
// anything long-lived inside the callback: whatever is built there is bound to a context
// this function cancels on the way out, and a subscription that dies that way looks exactly
// like a camera that stops answering after a minute.
func runOneShot(ctx context.Context, run func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, oneShotDeadline)
	defer cancel()
	return run(ctx)
}

var (
	ErrMissingSubcommand = errors.New("missing sub-command")
)

// shutdownSignals is what a graceful stop can be built on.
//
// os.Kill was listed here and could never fire: os/signal documents that "SIGKILL and
// SIGSTOP may not be caught by a program", so the entry cost nothing while SIGTERM -- what
// `kill <pid>`, systemd and a container stop actually send -- reached nothing, and a run
// under a one-minute deadline kept probing until the deadline instead of stopping.
//
// A package-level var rather than a literal in the call, so a test can assert the set
// without sending signals to the test binary.
var shutdownSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

// newRootCommand builds the whole command tree.
//
// Separate from main() so that a test can inspect the tree without running it -- the
// PersistentPreRunE below is an invariant cobra does not enforce -- and so that main()
// stays what it says it is: a context, a tree and Execute.
func newRootCommand(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		// cobra takes the first word of Use as the command name and builds every usage
		// line from it, so "main" — the package name — had the help of every subcommand
		// telling the operator to run a binary that does not exist.
		Use:   "onvif-cli",
		Short: "OnVif command Line Interface",
		Long:  "CLI Client for OnVif devices",
		// Now that there are flags there are usage errors, and cobra already prints
		// "Error: ..." with the usage block; the Fatal below would say it a second time.
		SilenceErrors: true,
		RunE:          func(cmd *cobra.Command, args []string) error { return ErrMissingSubcommand },
	}

	// --basedir is the exception to the rule below, and the comment there says which side
	// of it a flag falls on: there is exactly one credentials store per process, so every
	// command must see the same value, and the coupling is the semantics rather than an
	// accident. Hence a persistent flag on the root, bound to a package-level variable.
	//
	// The default stays empty on purpose. It is what distinguishes "not named" from
	// "named explicitly", and printing one directory as a default would describe the
	// search chain wrongly; the chain belongs in the help text.
	// The backticks are not decoration: pflag takes the quoted word as the value name, so
	// the help reads "--basedir DIR" rather than "--basedir string", which is what
	// README.md documents.
	cmd.PersistentFlags().StringVar(&baseDir, "basedir", "",
		"base directory of the per-camera credential files, read from "+
			"`DIR`/credentials/*.json (default $ONVIF_BASEDIR, else ~/.onvif, else /etc/onvif)")

	// Counted rather than named: -vv is how every tool an operator already uses spells
	// "more", and it needs no table of level names to remember or to keep in step with
	// zerolog's. Persistent for the same reason --basedir is -- one process, one level.
	cmd.PersistentFlags().CountVarP(&verbosity, "verbose", "v",
		"print more on stderr: -v what the tool is doing, -vv what it loaded and which "+
			"credentials it chose, -vvv the discovery probe and every per-call failure. "+
			"Warnings and errors are printed either way")

	// The credentials are read here rather than at process start because the flag does not
	// exist until cobra has parsed, which happens inside Execute(). This hook runs once,
	// before the RunE of whichever leaf was invoked, so a mistyped path or a malformed file
	// stops the run before a single camera is contacted -- and its error travels the path
	// every other error already travels, out of Execute() and into the Fatal below.
	//
	// It must stay on the root and nowhere else: cobra runs only the closest hook in the
	// chain, so one on `dump` would shadow this and leave the resolver nil for its leaves.
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Parsing has already succeeded, so nothing that follows is a usage error and the
		// usage block would only bury the message.
		cmd.SilenceUsage = true

		// Before anything that logs, including the credential loading below. The global
		// level gates every zerolog logger in the process, so this reaches sdk's as well
		// as ours -- which is the point, since the per-call failures sdk swallows are
		// what -vvv is for.
		zerolog.SetGlobalLevel(verbosityLevel(verbosity))

		r, err := buildResolver(baseDir)
		if err != nil {
			return err
		}
		resolver = r
		return nil
	}

	// Each command owns its variable: cobra binds a flag to an address, and sharing one
	// address across two commands would couple their defaults for no gain. Reading the
	// value back with Flags().GetBool would instead hand us an error to ignore. A flag
	// that genuinely names one process-wide policy, such as --basedir above, is the case
	// on the other side of that line.
	var discoverAll, streamsAll bool

	cmdDiscover := &cobra.Command{
		Use:     "discover",
		Aliases: []string{"find", "crawl", "probe"},
		Short:   "Discover the local cameras",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOneShot(ctx, func(ctx context.Context) error {
				return discover(ctx, discoverOptions{allInterfaces: discoverAll})
			})
		},
	}
	cmdDiscover.Flags().BoolVarP(&discoverAll, "all", "a", false, allInterfacesHelp)

	cmdStreams := &cobra.Command{
		Use:     "streams",
		Aliases: []string{"stream"},
		Short:   "Print the stream URL for the cameras locally discovered",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOneShot(ctx, func(ctx context.Context) error {
				return discover(ctx, discoverOptions{streams: true, allInterfaces: streamsAll})
			})
		},
	}
	cmdStreams.Flags().BoolVarP(&streamsAll, "all", "a", false, allInterfacesHelp)

	cmdSubscribe := &cobra.Command{
		Use: "subscribe TARGET [TARGET...]",
		// `watch` is what a person types; `pull` is ONVIF's own word for the mechanism
		// (Core section 9.1). Not `events`: `dump event` already answers to that, and
		// `onvif-cli events` meaning "stream them" beside `onvif-cli dump event` meaning
		// "describe the service" is a collision an operator would hit once and then
		// remember the wrong way round.
		Aliases: []string{"watch", "pull"},
		Short:   "Stream the events of the given cameras, one JSON object per line",
		Long: "Stream the events of the given cameras, one JSON object per line.\n\n" +
			targetHelp + "\n\n" + subscribeHelp,
		Example: "  onvif-cli subscribe 192.168.1.70:80\n" +
			"  onvif-cli subscribe urn:uuid:00000700-0013-0008-0203-ec71db76e907 192.168.1.71:80\n" +
			"  onvif-cli discover | awk '$3 != \"-\" { print $3 }' | xargs onvif-cli subscribe",
		// At least one, and no upper bound: the fleet form is the point of the command, and
		// no argument does not mean "every camera" -- see subscribeHelp on why there is no
		// --all here.
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { return subscribe(ctx, args) },
	}

	// "all" is deliberately not an alias here, though it reads like one: a child of this
	// command already owns that name. With both, `onvif-cli all` resolved to the bare parent
	// -- an error -- while `onvif-cli dump all` dumped, so one word meant two things.
	cmdDump := &cobra.Command{
		Use:     "dump",
		Aliases: []string{"detail", "details"},
		Short:   "Dump the configuration of the given camera",
		Args:    cobra.NoArgs,
		RunE:    func(cmd *cobra.Command, args []string) error { return ErrMissingSubcommand },
	}

	// Every leaf takes the same single positional and differs only in what it prints, so
	// the argument handling -- which now has to tell an address from an identifier, and
	// probe the LAN for the latter -- is written once rather than nine times.
	cmdDumpAll := dumpCommand(ctx, "all", []string{"full"},
		"Dump the configuration of the given camera", dumpAll)
	cmdDumpDescr := dumpCommand(ctx, "descriptor", []string{"minimal", "mini"},
		"Dump a general descriptor of the given camera", dumpDescriptor)
	cmdDumpMedia := dumpCommand(ctx, "media", nil,
		"Dump the information related to the Media service of the camera", dumpMedia)
	cmdDumpPtz := dumpCommand(ctx, "ptz", []string{"PTZ"},
		"Dump the information related to the PTZ service of the camera", dumpPTZ)
	cmdDumpEvents := dumpCommand(ctx, "event", []string{"events", "evt"},
		"Dump the information related to the Events service of the camera", dumpEvents)
	cmdDumpProfiles := dumpCommand(ctx, "profile", []string{"profiles", "prof"},
		"Dump the information related to the Profiles of the camera", dumpProfiles)
	cmdDumpDevice := dumpCommand(ctx, "device", []string{"devices", "dev"},
		"Dump the information related to the core Device service of the camera", dumpDevice)

	cmdDump.AddCommand(cmdDumpDescr, cmdDumpAll)
	cmdDump.AddCommand(cmdDumpMedia, cmdDumpPtz, cmdDumpEvents, cmdDumpProfiles, cmdDumpDevice)
	cmd.AddCommand(cmdDiscover, cmdStreams, cmdSubscribe, cmdDump)

	return cmd
}

func main() {
	// Signal handling and nothing else. The one-minute deadline that used to wrap this
	// context bounded the whole process, which was right while every command was one-shot;
	// it now belongs to the commands that have an end of their own. See runOneShot.
	ctx, stop := signal.NotifyContext(context.Background(), shutdownSignals...)
	defer stop()

	// A second signal is the operator's way out, and `subscribe` is why it is needed. It
	// writes to stdout synchronously under a lock, so an Encode blocked on a pipe nobody is
	// reading is not interruptible by a context at all: no goroutine is in a select to notice
	// the first signal, and `subscribe ... | less` left unread hangs. NotifyContext keeps its
	// handler registered after it fires, so without this the second Ctrl-C is swallowed too
	// and only SIGKILL ends the run. Restoring the default disposition is the usual answer:
	// the first signal asks for a graceful stop, the second takes it.
	//
	// The goroutine AfterFunc starts is deliberately outside any WaitGroup -- it is the one
	// place in this tool where that is right, because its whole purpose is to run when
	// nothing else can. It must stay a call to stop and nothing else: it can fire while
	// recordWriter holds its mutex, so anything here that took a lock of ours would deadlock
	// the escape hatch.
	context.AfterFunc(ctx, stop)

	// The default level, established before Execute rather than only in the root's
	// PersistentPreRunE, because cobra skips that hook on its help and completion paths: a
	// plain `onvif-cli --help` otherwise ran at zerolog's own default and ended with a
	// timestamped DBG line under the usage block. The hook still runs for every real
	// command, and only ever lowers this.
	zerolog.SetGlobalLevel(verbosityLevel(0))

	cmd := newRootCommand(ctx)

	if err := cmd.Execute(); err != nil {
		Logger.Fatal().Err(err).Msg("Aborting")
	} else {
		// A lifecycle marker, not a result: a run that worked says so by what it printed
		// on stdout and by its exit status. At debug it is still there for anyone
		// following the sequence with -vv.
		Logger.Debug().Msg("Exiting")
	}
}
