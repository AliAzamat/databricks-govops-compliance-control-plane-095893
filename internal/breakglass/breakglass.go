// Package breakglass grants a TIME-BOXED waiver for a single control, so an
// operator can promote past a known, accepted non-compliance — but only with a
// justification, an approver, an expiry, and a MANDATORY ledger entry. If the
// waiver cannot be recorded, it is not granted.
package breakglass

import (
	"errors"
	"time"

	"github.com/khwajalabs/govops/internal/ledger"
)

type Waiver struct {
	ControlID     string
	Actor         string
	Approver      string // must differ from Actor — two-person rule
	Justification string
	GrantedAt     time.Time
	ExpiresAt     time.Time
}

// Active reports whether the waiver still covers the control at time `now`.
// An expired waiver is no waiver — the control snaps back to fail-closed.
func (w Waiver) Active(now time.Time) bool {
	return now.Before(w.ExpiresAt)
}

var (
	ErrSelfApproval    = errors.New("break-glass requires a different approver")
	ErrNoJustification = errors.New("break-glass requires a justification")
	ErrNotTimeBoxed    = errors.New("break-glass requires an expiry in the future")
)

// Authorize validates the waiver and records it in the ledger BEFORE returning
// it. It returns a usable waiver only if the request is valid AND the ledger
// entry was written — the record of the exception exists before the exception
// is usable.
func Authorize(w Waiver, snapshotAdr string, lg *ledger.Ledger) (Waiver, error) {
	if w.Approver == "" || w.Approver == w.Actor {
		return Waiver{}, ErrSelfApproval
	}
	if w.Justification == "" {
		return Waiver{}, ErrNoJustification
	}
	if !w.ExpiresAt.After(w.GrantedAt) {
		return Waiver{}, ErrNotTimeBoxed
	}
	// Ledger FIRST — the waiver is on the record before it can be used to promote.
	lg.Append(ledger.Entry{
		Kind:        ledger.KindWaiver,
		Actor:       w.Actor,
		SnapshotAdr: snapshotAdr,
		Detail: "control=" + w.ControlID +
			" approver=" + w.Approver +
			" expires=" + w.ExpiresAt.UTC().Format(time.RFC3339) +
			" justification=" + w.Justification,
	})
	return w, nil
}
