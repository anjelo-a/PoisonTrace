package runs

import "time"

type Counters struct {
	WalletsRequested            int
	WalletsProcessed            int
	WalletsFailed               int
	WalletsSkipped              int
	TransactionsFetched         int
	TransactionsInserted        int
	TransactionsLinked          int
	TransactionsFailedNormalize int
	OwnerUnresolvedCount        int
	DecimalsUnresolvedCount     int
	CounterpartiesCreated       int
	CounterpartiesUpdated       int
	PoisoningCandidatesInserted int
	RetryExhaustedCount         int
}

func IsPartial(progressPersisted bool, hardFailure bool) bool {
	return progressPersisted && hardFailure
}

func BuildWindow(scanStart, scanEnd time.Time, baselineLookbackDays int) WalletSyncWindow {
	baseEnd := scanStart.UTC()
	baseStart := baseEnd.AddDate(0, 0, -baselineLookbackDays).UTC()
	return WalletSyncWindow{
		BaselineStart: baseStart,
		BaselineEnd:   baseEnd,
		ScanStart:     scanStart.UTC(),
		ScanEnd:       scanEnd.UTC(),
	}
}
