package main

import (
	"fmt"
	"time"

	"github.com/khwajalabs/govops/internal/breakglass"
	"github.com/khwajalabs/govops/internal/catalog"
	"github.com/khwajalabs/govops/internal/drift"
	"github.com/khwajalabs/govops/internal/evidence"
	"github.com/khwajalabs/govops/internal/ledger"
	"github.com/khwajalabs/govops/internal/promote"
	"github.com/khwajalabs/govops/internal/snapshot"
)

// App is the operator surface: subcommands snapshot, gate, drift, waive, and
// verify-ledger. It holds the wiring so each subcommand is a thin, typed call.
// Exit codes are stable so automation can branch on them:
//   0 ok/compliant   2 non-compliant/blocked/drift   1 operational error
type App struct {
	Baseline  catalog.Baseline
	Registry  evidence.Registry
	Ledger    *ledger.Ledger
	KnownGood *snapshot.Snapshot   // last known-good, for drift (nil until first set)
	Waivers   []breakglass.Waiver
	Now       func() time.Time
}

// Snapshot takes a point-in-time snapshot, records it in the ledger, and prints
// its address. Taking a snapshot is a pure read of the platform, but the fact
// that it was taken IS a compliance event, so it is ledgered.
func (a *App) Snapshot() (snapshot.Snapshot, int) {
	snap := snapshot.Take(a.Baseline, a.Registry, a.Now())
	ok, failing := snap.Satisfies(a.Baseline)
	a.Ledger.Append(ledger.Entry{
		Kind: ledger.KindSnapshot, Actor: "operator", SnapshotAdr: snap.Address(),
		Detail: fmt.Sprintf("compliant=%v failing=%d", ok, len(failing)),
	})
	fmt.Printf("snapshot %s (compliant=%v, %d failing)\n", snap.Address(), ok, len(failing))
	return snap, exitForCompliance(ok)
}

// Gate reports whether a release may promote. It is report-first: it prints the
// decision and NEVER mutates the platform. A blocked promotion is exit 2.
func (a *App) Gate(release string, snap snapshot.Snapshot) int {
	d := promote.Gate(release, snap, a.Baseline, a.Waivers, a.Now())
	a.Ledger.Append(ledger.Entry{
		Kind: ledger.KindPromotion, Actor: "operator", SnapshotAdr: d.SnapshotAdr,
		Detail: fmt.Sprintf("release=%s allowed=%v blocking=%d waived=%d",
			release, d.Allowed, len(d.Blocking), len(d.Waived)),
	})
	fmt.Println(d.Reason())
	if d.Allowed {
		return 0
	}
	return 2
}

// Drift compares a fresh snapshot against the known-good and reports regressions.
// A regression is exit 2 so an on-call watcher can alarm on it.
func (a *App) Drift(current snapshot.Snapshot) int {
	if a.KnownGood == nil {
		fmt.Println("no known-good snapshot set; nothing to diff against")
		return 1
	}
	rep := drift.Diff(*a.KnownGood, current)
	if !rep.HasRegression() {
		fmt.Printf("no drift: %d control(s) unchanged or recovered\n", len(rep.Changes))
		return 0
	}
	fmt.Printf("DRIFT: %d control(s) regressed\n", len(rep.Regressions))
	for _, c := range rep.Regressions {
		fmt.Printf("  - %s: %s -> %s\n", c.ControlID, c.Was, c.Now)
	}
	return 2
}

// Waive grants a time-boxed break-glass waiver. It refuses (exit 1) if the waiver
// is invalid or cannot be recorded — privilege is never granted unrecorded.
func (a *App) Waive(w breakglass.Waiver, snapAdr string) int {
	granted, err := breakglass.Authorize(w, snapAdr, a.Ledger)
	if err != nil {
		fmt.Printf("waiver refused: %v\n", err)
		return 1
	}
	a.Waivers = append(a.Waivers, granted)
	fmt.Printf("waiver granted for %s until %s\n", granted.ControlID, granted.ExpiresAt.UTC().Format(time.RFC3339))
	return 0
}

// VerifyLedger verifies the whole compliance ledger end to end. Exit 2 if the
// chain is broken at any index — a tampered evidence trail is a hard failure.
func (a *App) VerifyLedger() int {
	if bad := a.Ledger.Verify(); bad != -1 {
		fmt.Printf("LEDGER TAMPERED: chain broken at entry %d\n", bad)
		return 2
	}
	fmt.Printf("ledger intact: %d entries verified\n", len(a.Ledger.Entries()))
	return 0
}

func exitForCompliance(ok bool) int {
	if ok {
		return 0
	}
	return 2
}
