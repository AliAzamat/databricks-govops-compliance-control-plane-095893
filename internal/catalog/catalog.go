// Package catalog defines the compliance controls and the baseline that selects
// which of them a running government-cloud platform must satisfy. Controls are
// DATA, not code paths — a FedRAMP-style catalog you can version and audit.
package catalog

// Control is one compliance requirement. It describes WHAT must be true and what
// evidence proves it — never HOW the platform achieves it.
type Control struct {
	ID           string `json:"id"`            // e.g. "SC-28" (protection at rest)
	Description  string `json:"description"`   // human-readable requirement
	Family       string `json:"family"`        // control family, e.g. "SC", "AC", "AU"
	RequiredEvid string `json:"required_evid"` // the evidence kind a collector must return
}

// Baseline is the declarative set of controls this platform is held to. It is
// the DESIRED compliance posture — the control plane's job is to prove reality
// matches it, continuously. Two baselines with the same controls are the same
// baseline; that is what the content address in step three relies on.
type Baseline struct {
	Name     string    `json:"name"`     // e.g. "gov-prod-moderate"
	Controls []Control `json:"controls"`
}

// GovProdModerate is a small FedRAMP-Moderate-style baseline for a gov-prod
// platform. Real baselines have hundreds of controls; the shape is identical.
func GovProdModerate() Baseline {
	return Baseline{
		Name: "gov-prod-moderate",
		Controls: []Control{
			{ID: "SC-28", Family: "SC", Description: "Data is encrypted at rest", RequiredEvid: "encryption-at-rest"},
			{ID: "IA-2", Family: "IA", Description: "Operators authenticate with MFA", RequiredEvid: "mfa-enforced"},
			{ID: "AU-2", Family: "AU", Description: "Auditable events are logged", RequiredEvid: "logging-enabled"},
		},
	}
}

// IDs returns the control IDs in the baseline, in declared order.
func (b Baseline) IDs() []string {
	ids := make([]string, len(b.Controls))
	for i, c := range b.Controls {
		ids[i] = c.ID
	}
	return ids
}
