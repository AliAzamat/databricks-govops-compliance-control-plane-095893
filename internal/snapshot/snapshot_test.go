package snapshot

import "testing"

func TestSnapshotReHashesDeterministically(t *testing.T) {
	s1, _ := compliantSnapshot()
	s2, _ := compliantSnapshot()
	if s1.Address() != s2.Address() {
		t.Fatalf("identical compliance facts must hash identically:\n %s\n %s",
			s1.Address(), s2.Address())
	}
}

func TestAddressIgnoresCollectedAt(t *testing.T) {
	s1, _ := compliantSnapshot()
	// Same facts, different wall-clock instant: the address must be unchanged
	// because CollectedAt and TakenAt are excluded from the content.
	s2, _ := compliantSnapshot()
	s2.TakenAt = s2.TakenAt.Add(90 * 60 * 1e9) // +90 minutes
	for i := range s2.Attestations {
		s2.Attestations[i].CollectedAt = s2.Attestations[i].CollectedAt.Add(90 * 60 * 1e9)
	}
	if s1.Address() != s2.Address() {
		t.Fatal("the content address must ignore collection time; it commits to facts")
	}
}
