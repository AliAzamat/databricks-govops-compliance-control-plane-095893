// Package evidence collects proof that a control holds RIGHT NOW. Each collector
// returns a typed Attestation. The cardinal rule is fail-closed: a control with
// no collector, or a collector that errors, is NOT compliant by default.
package evidence

import (
	"fmt"
	"time"

	"github.com/khwajalabs/govops/internal/catalog"
)

// Status is the outcome of checking one control. Unknown is distinct from Fail:
// "we could not get evidence" is its own state, and it is treated as non-compliant.
type Status string

const (
	Pass    Status = "pass"
	Fail    Status = "fail"
	Unknown Status = "unknown" // no evidence obtained — fails closed
)

// Attestation is the typed result of checking one control at one moment. It is
// the atom the snapshot, gate, drift, and ledger all reason about.
type Attestation struct {
	ControlID   string    `json:"control_id"`
	Status      Status    `json:"status"`
	Evidence    string    `json:"evidence"`     // what was observed
	CollectedAt time.Time `json:"collected_at"`
}

// Compliant is the ONLY thing that counts as satisfying a control. Note it is
// true for Pass and nothing else — Unknown and Fail are both non-compliant.
func (a Attestation) Compliant() bool { return a.Status == Pass }

// Collector produces evidence for exactly one kind. A collector declares the
// evidence Kind it serves so the registry can pair it with a control.
type Collector interface {
	Kind() string                       // the RequiredEvid it can satisfy
	Collect() (status Status, evidence string, err error)
}

// Registry maps an evidence kind to the collector that produces it.
type Registry map[string]Collector

// Register adds a collector under its declared kind.
func (r Registry) Register(c Collector) { r[c.Kind()] = c }

// Attest checks a single control. This is where deny-by-default lives:
//   - no collector for the control's required evidence -> Unknown (fails closed)
//   - the collector errors                             -> Unknown (fails closed)
//   - otherwise the collector's own Pass/Fail verdict
func (r Registry) Attest(c catalog.Control, now time.Time) Attestation {
	col, ok := r[c.RequiredEvid]
	if !ok {
		return Attestation{
			ControlID:   c.ID,
			Status:      Unknown,
			Evidence:    fmt.Sprintf("no collector registered for evidence %q", c.RequiredEvid),
			CollectedAt: now,
		}
	}
	status, ev, err := col.Collect()
	if err != nil {
		return Attestation{
			ControlID:   c.ID,
			Status:      Unknown,
			Evidence:    fmt.Sprintf("collector error: %v", err),
			CollectedAt: now,
		}
	}
	return Attestation{ControlID: c.ID, Status: status, Evidence: ev, CollectedAt: now}
}
