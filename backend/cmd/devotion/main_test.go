package main

import (
	"sort"
	"testing"
)

// TestDispatcher_EightSubcommandsRegistered_GateI verifies the registry holds
// exactly the eight known subcommands. Subcommands are one-shot processes, not
// runtime services, so this does not affect the two-service count; the test
// exists to catch an accidental extra process disguised as a subcommand.
func TestDispatcher_EightSubcommandsRegistered_GateI(t *testing.T) {
	want := []string{
		"admin:create",
		"health:check",
		"reset:test-data",
		"seed:master-data",
		"seed:regions",
		"seed:test-data",
		"serve",
		"user:verify",
	}

	got := make([]string, 0, len(commands))
	for name := range commands {
		got = append(got, name)
	}
	sort.Strings(got)

	if len(got) != len(want) {
		t.Fatalf("jumlah subcommand = %d, ingin %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("subcommand[%d] = %q, ingin %q", i, got[i], want[i])
		}
	}
}
