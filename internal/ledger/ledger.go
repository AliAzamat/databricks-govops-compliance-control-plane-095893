// Package ledger is an append-only, hash-chained log of COMPLIANCE EVENTS: every
// snapshot taken and every promotion decision made. Each entry commits to the
// previous entry's hash, so altering any past entry breaks every hash after it —
// tampering is detectable, not preventable.
package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// Kind distinguishes the compliance events the ledger records.
type Kind string

const (
	KindSnapshot  Kind = "snapshot"        // a compliance snapshot was taken
	KindPromotion Kind = "promotion"       // a promotion was allowed or blocked
	KindWaiver    Kind = "break-glass-waiver"
)

// Entry is one compliance event. It references a snapshot by its content ADDRESS
// (not a copy) — the snapshot's own hash grounds the entry, mirroring how the
// exemplar's audit entries referenced specs by content address.
type Entry struct {
	Seq         int       `json:"seq"`
	Time        time.Time `json:"time"`
	Kind        Kind      `json:"kind"`
	Actor       string    `json:"actor"`
	SnapshotAdr string    `json:"snapshot_adr"` // content address of the snapshot involved
	Detail      string    `json:"detail"`       // decision, control id, justification, etc.
	PrevHash    string    `json:"prev_hash"`    // hash of entry Seq-1
	Hash        string    `json:"hash"`         // hash of THIS entry (excluding Hash)
}

type Ledger struct {
	entries []Entry
	now     func() time.Time
}

// New returns a ledger with an injectable clock so tests are deterministic.
func New(now func() time.Time) *Ledger {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Ledger{now: now}
}

// Append adds an entry, chaining it to the previous entry's hash.
func (l *Ledger) Append(e Entry) Entry {
	e.Seq = len(l.entries)
	e.Time = l.now()
	if len(l.entries) > 0 {
		e.PrevHash = l.entries[len(l.entries)-1].Hash
	} else {
		e.PrevHash = "genesis"
	}
	e.Hash = hashEntry(e)
	l.entries = append(l.entries, e)
	return e
}

func hashEntry(e Entry) string {
	e.Hash = "" // the hash excludes the Hash field itself
	b, _ := json.Marshal(e)
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Verify walks the chain and returns the first sequence number whose hash or
// prev-link does not check out. -1 means the whole chain is intact.
func (l *Ledger) Verify() int {
	prev := "genesis"
	for i, e := range l.entries {
		if e.PrevHash != prev {
			return i // chain link broken
		}
		if e.Hash != hashEntry(e) {
			return i // entry content was altered
		}
		prev = e.Hash
	}
	return -1
}

func (l *Ledger) Entries() []Entry { return l.entries }
