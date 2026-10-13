// Package drift detects a compliance REGRESSION: a control that was compliant in
// the last known-good snapshot and is no longer compliant now. This is the
// CONTINUOUS half of GovOps — the platform was proven compliant once; drift is
// how you notice it silently stopped being so.
package drift

import (
	"github.com/khwajalabs/govops/internal/evidence"
	"github.com/khwajalabs/govops/internal/snapshot"
)

// Change is one control's movement between the known-good snapshot and the current
// one. Regressed is the field that matters: a pass that became non-compliant.
type Change struct {
	ControlID string
	Was       evidence.Status
	Now       evidence.Status
	Regressed bool // was Pass, now not Pass
	Recovered bool // was not Pass, now Pass
}

// Report is the full diff plus a fast "did anything regress" answer.
type Report struct {
	Changes    []Change
	Regressions []Change // subset of Changes where Regressed is true
}

// HasRegression is the alarm the operator watches — true if ANY control that was
// compliant is no longer compliant.
func (r Report) HasRegression() bool { return len(r.Regressions) > 0 }

// Diff compares the current snapshot against a known-good one. It reports, per
// control in the known-good snapshot, whether it regressed or recovered. A
// control present in known-good but MISSING from current is treated as a
// regression to Unknown — dropping a control out of measurement is a regression,
// not a silent recovery.
func Diff(knownGood, current snapshot.Snapshot) Report {
	cur := index(current)
	var rep Report
	for _, was := range knownGood.Attestations {
		now, present := cur[was.ControlID]
		nowStatus := evidence.Unknown
		if present {
			nowStatus = now.Status
		}
		wasCompliant := was.Status == evidence.Pass
		nowCompliant := nowStatus == evidence.Pass
		ch := Change{
			ControlID: was.ControlID,
			Was:       was.Status,
			Now:       nowStatus,
			Regressed: wasCompliant && !nowCompliant,
			Recovered: !wasCompliant && nowCompliant,
		}
		rep.Changes = append(rep.Changes, ch)
		if ch.Regressed {
			rep.Regressions = append(rep.Regressions, ch)
		}
	}
	return rep
}

func index(s snapshot.Snapshot) map[string]evidence.Attestation {
	m := make(map[string]evidence.Attestation, len(s.Attestations))
	for _, a := range s.Attestations {
		m[a.ControlID] = a
	}
	return m
}
