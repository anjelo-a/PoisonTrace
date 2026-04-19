package pipeline

const (
	ingestionFinalizeTimeoutSeconds = 3
	// Keep the wallet lock alive past the wallet timeout while finalize/release tail work completes.
	walletFinalizeTimeoutSeconds = 3
	walletLockReleaseTimeoutSecs = 2
	walletLockTTLTailSeconds     = walletFinalizeTimeoutSeconds + walletLockReleaseTimeoutSecs + 1
)
