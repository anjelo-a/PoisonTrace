package storage

import (
	"context"
	"database/sql/driver"
	"regexp"
	"strings"
	"testing"
	"time"

	"poisontrace/internal/runs"

	"github.com/DATA-DOG/go-sqlmock"
)

type anyTime struct{}

func (a anyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

func newMockStore(t *testing.T) (*PostgresStore, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	return NewPostgresStore(db), mock, func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet sql expectations: %v", err)
		}
		_ = db.Close()
	}
}

func TestFinalizeIngestionRunReturnsNotFoundOnZeroRows(t *testing.T) {
	t.Parallel()
	store, mock, done := newMockStore(t)
	defer done()

	counters := runs.Counters{WalletsRequested: 1}
	mock.ExpectExec(regexp.QuoteMeta("UPDATE ingestion_runs")).
		WithArgs(
			int64(99),
			runs.RunStatusSucceeded,
			anyTime{},
			counters.WalletsRequested,
			counters.WalletsProcessed,
			counters.WalletsFailed,
			counters.WalletsSkipped,
			counters.TruncationWalletCount,
			counters.TruncationWalletRate,
			counters.TransactionsFetched,
			counters.TransactionsInserted,
			counters.TransactionsLinked,
			counters.TransactionsFailedNormalize,
			counters.OwnerUnresolvedCount,
			counters.DecimalsUnresolvedCount,
			counters.CounterpartiesCreated,
			counters.CounterpartiesUpdated,
			counters.PoisoningCandidatesInserted,
			counters.RetryExhaustedCount,
			nil,
		).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := store.FinalizeIngestionRun(context.Background(), 99, runs.RunStatusSucceeded, time.Now().UTC(), counters, "")
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestUpdateWalletSyncProgressClearsUnknownGateReasonWhenComplete(t *testing.T) {
	t.Parallel()
	store, mock, done := newMockStore(t)
	defer done()

	progress := WalletSyncProgress{
		BaselineComplete:      true,
		IncompleteWindow:      false,
		UnknownGateReason:     "unknown_required_gates:incomplete_window",
		TruncationReason:      "",
		TransactionsFetched:   10,
		TransactionsInserted:  9,
		TransactionsLinked:    8,
		CounterpartiesCreated: 1,
		CounterpartiesUpdated: 2,
	}

	mock.ExpectExec(regexp.QuoteMeta("UPDATE wallet_sync_runs")).
		WithArgs(
			int64(7),
			progress.BaselineComplete,
			progress.IncompleteWindow,
			nil,
			nil,
			progress.TransactionsFetched,
			progress.TransactionsInserted,
			progress.TransactionsLinked,
			progress.TransactionsFailedNormalize,
			progress.CounterpartiesCreated,
			progress.CounterpartiesUpdated,
			progress.PoisoningCandidatesInserted,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.UpdateWalletSyncProgress(context.Background(), 7, progress); err != nil {
		t.Fatalf("update wallet sync progress: %v", err)
	}
}

func TestFinalizeWalletSyncRunClearsUnknownReasonWhenNotIncomplete(t *testing.T) {
	t.Parallel()
	store, mock, done := newMockStore(t)
	defer done()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE wallet_sync_runs")).
		WithArgs(
			int64(4),
			runs.WalletStatusSucceeded,
			anyTime{},
			false,
			nil,
			nil,
			nil,
			nil,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := store.FinalizeWalletSyncRun(
		context.Background(),
		4,
		runs.WalletStatusSucceeded,
		time.Now().UTC(),
		false,
		"unknown_required_gates:should_be_dropped",
		"",
		"",
		"",
	)
	if err != nil {
		t.Fatalf("finalize wallet sync run: %v", err)
	}
}

func TestNewHolderTokenRandomFormat(t *testing.T) {
	t.Parallel()

	first, err := newHolderToken()
	if err != nil {
		t.Fatalf("newHolderToken first: %v", err)
	}
	second, err := newHolderToken()
	if err != nil {
		t.Fatalf("newHolderToken second: %v", err)
	}
	if first == second {
		t.Fatalf("expected random tokens, got identical value %q", first)
	}

	pattern := regexp.MustCompile(`^rnd:[a-f0-9]{32}$`)
	if !pattern.MatchString(first) {
		t.Fatalf("unexpected first token format: %q", first)
	}
	if !pattern.MatchString(second) {
		t.Fatalf("unexpected second token format: %q", second)
	}
}
