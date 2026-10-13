package snapshot

import (
	"time"

	"github.com/khwajalabs/govops/internal/catalog"
	"github.com/khwajalabs/govops/internal/evidence"
)

// fullyCompliantRegistry registers a passing collector for every control kind in
// the gov-prod-moderate baseline. Tests perturb individual collectors from here.
func fullyCompliantRegistry() evidence.Registry {
	r := evidence.Registry{}
	r.Register(evidence.EncryptionAtRest{AllVolumesEncrypted: true})
	r.Register(evidence.MFAEnforced{OperatorsWithoutMFA: 0})
	r.Register(evidence.LoggingEnabled{AuditPipelineHealthy: true})
	return r
}

var fixedTime = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

func compliantSnapshot() (Snapshot, catalog.Baseline) {
	b := catalog.GovProdModerate()
	return Take(b, fullyCompliantRegistry(), fixedTime), b
}
