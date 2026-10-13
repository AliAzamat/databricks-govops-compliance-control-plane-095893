package promote

import (
	"testing"
	"time"

	"github.com/khwajalabs/govops/internal/catalog"
	"github.com/khwajalabs/govops/internal/evidence"
	"github.com/khwajalabs/govops/internal/snapshot"
)

func registry(encrypted bool) evidence.Registry {
	r := evidence.Registry{}
	r.Register(evidence.EncryptionAtRest{AllVolumesEncrypted: encrypted})
	r.Register(evidence.MFAEnforced{OperatorsWithoutMFA: 0})
	r.Register(evidence.LoggingEnabled{AuditPipelineHealthy: true})
	return r
}

var when = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

func TestCompliantSnapshotAllowsPromotion(t *testing.T) {
	b := catalog.GovProdModerate()
	snap := snapshot.Take(b, registry(true), when)

	d := Gate("rel-1", snap, b, nil, when)
	if !d.Allowed {
		t.Fatalf("a fully compliant snapshot must promote; blocked by %d control(s)", len(d.Blocking))
	}
}

func TestNonCompliantSnapshotBlocksPromotion(t *testing.T) {
	b := catalog.GovProdModerate()
	// Encryption-at-rest regresses: SC-28 will fail.
	snap := snapshot.Take(b, registry(false), when)

	d := Gate("rel-2", snap, b, nil, when)
	if d.Allowed {
		t.Fatal("a non-compliant snapshot must be blocked (fail closed)")
	}
	if len(d.Blocking) != 1 || d.Blocking[0].ControlID != "SC-28" {
		t.Fatalf("expected SC-28 to be the sole blocker, got %+v", d.Blocking)
	}
	if d.Blocking[0].Status != evidence.Fail {
		t.Fatalf("SC-28 should block with status fail, got %s", d.Blocking[0].Status)
	}
}
