package drift

import (
	"testing"
	"time"

	"github.com/khwajalabs/govops/internal/catalog"
	"github.com/khwajalabs/govops/internal/evidence"
	"github.com/khwajalabs/govops/internal/snapshot"
)

func reg(logging bool) evidence.Registry {
	r := evidence.Registry{}
	r.Register(evidence.EncryptionAtRest{AllVolumesEncrypted: true})
	r.Register(evidence.MFAEnforced{OperatorsWithoutMFA: 0})
	r.Register(evidence.LoggingEnabled{AuditPipelineHealthy: logging})
	return r
}

var t0 = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

func TestDriftDetectsRegression(t *testing.T) {
	b := catalog.GovProdModerate()
	knownGood := snapshot.Take(b, reg(true), t0)   // logging healthy: AU-2 passes
	current := snapshot.Take(b, reg(false), t0)     // logging broke: AU-2 fails

	rep := Diff(knownGood, current)
	if !rep.HasRegression() {
		t.Fatal("a control that went pass -> fail must be a regression")
	}
	if len(rep.Regressions) != 1 || rep.Regressions[0].ControlID != "AU-2" {
		t.Fatalf("expected AU-2 as the sole regression, got %+v", rep.Regressions)
	}
	if rep.Regressions[0].Was != evidence.Pass || rep.Regressions[0].Now != evidence.Fail {
		t.Fatalf("AU-2 should read pass -> fail, got %s -> %s",
			rep.Regressions[0].Was, rep.Regressions[0].Now)
	}
}

func TestNoRegressionWhenStable(t *testing.T) {
	b := catalog.GovProdModerate()
	knownGood := snapshot.Take(b, reg(true), t0)
	current := snapshot.Take(b, reg(true), t0)
	if Diff(knownGood, current).HasRegression() {
		t.Fatal("an unchanged, compliant platform must show no regression")
	}
}
