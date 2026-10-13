package evidence

// This file holds the concrete collectors. Each is pluggable: it satisfies the
// Collector interface and declares the evidence Kind it produces. In production
// a collector would query a real system (KMS, the IdP, the log pipeline); here
// they read an injected fact so tests can drive any posture deterministically.

// EncryptionAtRest proves control SC-28.
type EncryptionAtRest struct{ AllVolumesEncrypted bool }

func (c EncryptionAtRest) Kind() string { return "encryption-at-rest" }
func (c EncryptionAtRest) Collect() (Status, string, error) {
	if c.AllVolumesEncrypted {
		return Pass, "all volumes report AES-256 at rest", nil
	}
	return Fail, "one or more volumes are unencrypted at rest", nil
}

// MFAEnforced proves control IA-2.
type MFAEnforced struct{ OperatorsWithoutMFA int }

func (c MFAEnforced) Kind() string { return "mfa-enforced" }
func (c MFAEnforced) Collect() (Status, string, error) {
	if c.OperatorsWithoutMFA == 0 {
		return Pass, "every operator account has MFA enrolled", nil
	}
	return Fail, "operators without MFA present", nil
}

// LoggingEnabled proves control AU-2.
type LoggingEnabled struct{ AuditPipelineHealthy bool }

func (c LoggingEnabled) Kind() string { return "logging-enabled" }
func (c LoggingEnabled) Collect() (Status, string, error) {
	if c.AuditPipelineHealthy {
		return Pass, "audit log pipeline is receiving events", nil
	}
	return Fail, "audit log pipeline is not receiving events", nil
}
