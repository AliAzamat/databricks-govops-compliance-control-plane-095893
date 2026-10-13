package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/khwajalabs/govops/internal/evidence"
)

// naiveAddress is a DELIBERATELY BROKEN address function: it hashes the
// attestations in whatever order they arrive, WITHOUT sorting. It exists only to
// demonstrate the bug the real Address() avoids.
func naiveAddress(s Snapshot) string {
	cs := make([]canonicalAtt, len(s.Attestations))
	for i, a := range s.Attestations {
		cs[i] = canonicalAtt{ControlID: a.ControlID, Status: a.Status, Evidence: a.Evidence}
	}
	// NOTE: no sort here — this is the bug.
	b, _ := json.Marshal(cs)
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// shuffled returns the same compliance facts with the attestation slice reversed —
// simulating collectors that ran in a different order (e.g. concurrently).
func shuffled(s Snapshot) Snapshot {
	rev := make([]evidence.Attestation, len(s.Attestations))
	for i, a := range s.Attestations {
		rev[len(s.Attestations)-1-i] = a
	}
	out := s
	out.Attestations = rev
	return out
}

func TestChaosOrderingBreaksNaiveHashButNotAddress(t *testing.T) {
	snap, _ := compliantSnapshot()
	perturbed := shuffled(snap)

	// Sanity: the two snapshots hold the SAME compliance facts, just reordered.
	if len(snap.Attestations) != len(perturbed.Attestations) {
		t.Fatal("perturbation must not add or drop attestations")
	}

	// The naive, unsorted hash DIVERGES under reordering — the same compliance
	// state gets two different "addresses". This is the bug.
	if naiveAddress(snap) == naiveAddress(perturbed) {
		t.Fatal("expected the unsorted hash to diverge under reordering; it did not")
	}

	// The real Address() sorts before hashing, so the SAME facts in a different
	// order produce the SAME address. This is the fix, and the whole reason an
	// auditor can trust a content address.
	if snap.Address() != perturbed.Address() {
		t.Fatalf("Address() must be order-independent:\n %s\n %s",
			snap.Address(), perturbed.Address())
	}
}
