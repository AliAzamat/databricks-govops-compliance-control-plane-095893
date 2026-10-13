// Package snapshot runs every control in a baseline through the evidence
// registry and captures a point-in-time attestation set — then content-addresses
// it so an auditor can verify EXACTLY what was true at that moment.
package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"github.com/khwajalabs/govops/internal/catalog"
	"github.com/khwajalabs/govops/internal/evidence"
)

// Snapshot is the compliance state of the platform at one instant: the baseline
// it was measured against and one Attestation per control.
type Snapshot struct {
	Baseline     string                    `json:"baseline"`
	TakenAt      time.Time                 `json:"taken_at"`
	Attestations []evidence.Attestation    `json:"attestations"`
}

// Take runs the whole baseline through the registry. `now` is injected so a
// snapshot is reproducible in tests — real callers pass time.Now().UTC().
func Take(b catalog.Baseline, reg evidence.Registry, now time.Time) Snapshot {
	atts := make([]evidence.Attestation, 0, len(b.Controls))
	for _, c := range b.Controls {
		atts = append(atts, reg.Attest(c, now))
	}
	return Snapshot{Baseline: b.Name, TakenAt: now, Attestations: atts}
}

// Address is the SHA-256 of the snapshot's canonical form: the baseline name and
// the attestations SORTED by control id, each reduced to (id,status,evidence).
// Two snapshots with the same compliance FACTS have the same address, regardless
// of the order collectors happened to run in — that is what an auditor verifies.
func (s Snapshot) Address() string {
	sum := sha256.Sum256(s.canonical())
	return "sha256:" + hex.EncodeToString(sum[:])
}

// canonicalAtt is the address-relevant projection of an attestation. Notably it
// EXCLUDES CollectedAt: the address commits to the compliance FACTS, not to the
// wall-clock instant, so re-hashing the same facts is stable.
type canonicalAtt struct {
	ControlID string          `json:"control_id"`
	Status    evidence.Status `json:"status"`
	Evidence  string          `json:"evidence"`
}

func (s Snapshot) canonical() []byte {
	cs := make([]canonicalAtt, len(s.Attestations))
	for i, a := range s.Attestations {
		cs[i] = canonicalAtt{ControlID: a.ControlID, Status: a.Status, Evidence: a.Evidence}
	}
	// Sort by control id so the SAME set of facts always serializes identically,
	// no matter what order the collectors ran in. Skip this and the same
	// compliance state hashes differently run to run — the address is worthless.
	sort.Slice(cs, func(i, j int) bool { return cs[i].ControlID < cs[j].ControlID })
	payload := struct {
		Baseline     string         `json:"baseline"`
		Attestations []canonicalAtt `json:"attestations"`
	}{Baseline: s.Baseline, Attestations: cs}
	b, _ := json.Marshal(payload)
	return b
}

// Satisfies reports whether the snapshot meets the baseline: every control has a
// COMPLIANT attestation. Missing or non-Pass -> not satisfied (fails closed).
func (s Snapshot) Satisfies(b catalog.Baseline) (bool, []evidence.Attestation) {
	byID := make(map[string]evidence.Attestation, len(s.Attestations))
	for _, a := range s.Attestations {
		byID[a.ControlID] = a
	}
	var failing []evidence.Attestation
	for _, c := range b.Controls {
		a, ok := byID[c.ID]
		if !ok || !a.Compliant() {
			if !ok {
				a = evidence.Attestation{ControlID: c.ID, Status: evidence.Unknown, Evidence: "control not in snapshot"}
			}
			failing = append(failing, a)
		}
	}
	return len(failing) == 0, failing
}
