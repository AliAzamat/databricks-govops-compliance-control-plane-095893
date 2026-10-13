package ledger

import "testing"

func TestVerifyDetectsForgedEntry(t *testing.T) {
	l := New(nil)
	l.Append(Entry{Kind: KindSnapshot, Actor: "op", SnapshotAdr: "sha256:aaa"})
	l.Append(Entry{Kind: KindWaiver, Actor: "op", SnapshotAdr: "sha256:bbb", Detail: "control=SC-28"})
	l.Append(Entry{Kind: KindPromotion, Actor: "op", SnapshotAdr: "sha256:ccc"})

	if bad := l.Verify(); bad != -1 {
		t.Fatalf("a fresh chain must verify, broke at %d", bad)
	}

	// Forge history: rewrite the waiver entry's detail to hide which control was
	// waived, keeping its stored hash.
	l.entries[1].Detail = "control=nothing-to-see-here"

	if bad := l.Verify(); bad != 1 {
		t.Fatalf("forging the waiver entry must be caught at index 1, got %d", bad)
	}
}
