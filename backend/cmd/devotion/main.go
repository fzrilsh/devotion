// Command devotion is the single binary for the Devotion backend. It dispatches
// to one of a fixed set of subcommands; there is no runtime process other than
// serve, so the two-service constraint (Gate I) holds.
package main

import (
	"context"
	"fmt"
	"os"
	"sort"
)

// command runs a subcommand with the arguments that follow its name.
type command func(ctx context.Context, args []string) error

// commands is the authoritative registry. Exactly these eight keys may exist;
// TestDispatcher_EightSubcommandsRegistered_GateI guards the count.
var commands = map[string]command{
	"serve":            runServe,
	"admin:create":     runAdminCreate,
	"seed:regions":     runSeedRegions,
	"seed:master-data": runSeedMasterData,
	"seed:test-data":   runSeedTestData,
	"reset:test-data":  runResetTestData,
	"user:verify":      runUserVerify,
	"health:check":     runHealthCheck,
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		usage(os.Stdout)
		return nil
	}
	name := args[0]
	cmd, ok := commands[name]
	if !ok {
		usage(os.Stderr)
		os.Exit(2)
	}
	return cmd(ctx, args[1:])
}

// usage prints the available subcommands, one per line, sorted for a stable
// listing.
func usage(w *os.File) {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Fprintln(w, "devotion: perintah tersedia:")
	for _, name := range names {
		fmt.Fprintln(w, "  "+name)
	}
}
