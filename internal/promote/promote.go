// Package promote gates a release moving from staging to gov-prod. A release is
// promoted ONLY if the current compliance snapshot satisfies the baseline. A
// non-compliant snapshot BLOCKS the promotion — fail closed — and the decision
// names exactly which controls stopped it and why.
package promote

import (
	"fmt"
	"strings"
	"time"

	"github.com/khwajalabs/govops/internal/breakglass"
	"github.com/khwajalabs/govops/internal/catalog"
	"github.com/khwajalabs/govops/internal/evidence"
	"github.com/khwajalabs/govops/internal/snapshot"
)

// Decision is the gate's verdict on a promotion. It is designed to be read by an
// operator: Allowed says yes/no, and Blocking lists the controls that failed so
// the operator sees WHICH control stopped the release, not just that one did.
type Decision struct {
	Release     string                 // the release/change being promoted
	FromEnv     string                 // "staging"
	ToEnv       string                 // "gov-prod"
	SnapshotAdr string                 // content address of the snapshot decided on
	Allowed     bool
	Blocking    []evidence.Attestation // the controls that blocked it (empty if allowed)
	Waived      []string               // control ids allowed only by an active waiver
}

// Reason renders a human-readable explanation an operator can act on.
func (d Decision) Reason() string {
	if d.Allowed {
		s := fmt.Sprintf("ALLOW %s -> %s (snapshot %s satisfies the baseline)", d.FromEnv, d.ToEnv, d.SnapshotAdr)
		if len(d.Waived) > 0 {
			s += fmt.Sprintf(" [waived: %s]", strings.Join(d.Waived, ","))
		}
		return s
	}
	var b strings.Builder
	fmt.Fprintf(&b, "BLOCK %s -> %s: %d control(s) not compliant\n", d.FromEnv, d.ToEnv, len(d.Blocking))
	for _, a := range d.Blocking {
		fmt.Fprintf(&b, "  - %s [%s] %s\n", a.ControlID, a.Status, a.Evidence)
	}
	return strings.TrimRight(b.String(), "\n")
}

// Gate decides whether `release` may promote given the current snapshot, the
// baseline, and any active waivers. A control that is non-compliant but covered
// by an ACTIVE (unexpired) waiver does not block — but it is recorded as waived,
// never silently ignored. It makes NO changes — a pure decision, safe to preview.
func Gate(release string, snap snapshot.Snapshot, base catalog.Baseline, waivers []breakglass.Waiver, now time.Time) Decision {
	_, failing := snap.Satisfies(base)
	active := activeWaivers(waivers, now)

	var blocking []evidence.Attestation
	var waived []string
	for _, a := range failing {
		if active[a.ControlID] {
			waived = append(waived, a.ControlID)
			continue // an active waiver covers this non-compliance
		}
		blocking = append(blocking, a)
	}
	return Decision{
		Release:     release,
		FromEnv:     "staging",
		ToEnv:       "gov-prod",
		SnapshotAdr: snap.Address(),
		Allowed:     len(blocking) == 0,
		Blocking:    blocking,
		Waived:      waived,
	}
}

func activeWaivers(waivers []breakglass.Waiver, now time.Time) map[string]bool {
	m := make(map[string]bool)
	for _, w := range waivers {
		if w.Active(now) {
			m[w.ControlID] = true
		}
	}
	return m
}
