package runs

import "time"

type RunStatus string

const (
	RunStatusRunning            RunStatus = "running"
	RunStatusSucceeded          RunStatus = "succeeded"
	RunStatusPartiallySucceeded RunStatus = "partially_succeeded"
	RunStatusFailed             RunStatus = "failed"
	RunStatusTimedOut           RunStatus = "timed_out"
	RunStatusCancelled          RunStatus = "cancelled"
)

type WalletStatus string

const (
	WalletStatusQueued         WalletStatus = "queued"
	WalletStatusRunning        WalletStatus = "running"
	WalletStatusSucceeded      WalletStatus = "succeeded"
	WalletStatusPartial        WalletStatus = "partial"
	WalletStatusFailed         WalletStatus = "failed"
	WalletStatusRateLimited    WalletStatus = "rate_limited"
	WalletStatusTimedOut       WalletStatus = "timed_out"
	WalletStatusSkippedInvalid WalletStatus = "skipped_invalid"
	WalletStatusSkippedBudget  WalletStatus = "skipped_budget"
)

type WalletSyncWindow struct {
	BaselineStart time.Time
	BaselineEnd   time.Time
	ScanStart     time.Time
	ScanEnd       time.Time
}
