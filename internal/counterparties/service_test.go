package counterparties

import (
	"testing"
	"time"

	"poisontrace/internal/transactions"
)

func TestMapWalletRelationUsesOwnerEndpoints(t *testing.T) {
	tr := transactions.NormalizedTransfer{
		SourceOwnerAddress:      "walletA",
		DestinationOwnerAddress: "walletB",
		SourceTokenAccount:      "tokenA",
		DestinationTokenAccount: "tokenB",
	}

	mapping, ok := MapWalletRelation("walletA", tr)
	if !ok {
		t.Fatal("expected sender relation for source owner")
	}
	if mapping.RelationType != RelationSender || mapping.CounterpartyAddress != "walletB" {
		t.Fatalf("unexpected mapping: %#v", mapping)
	}
}

func TestMapWalletRelationSkipsSelfTransferAndUnresolved(t *testing.T) {
	self := transactions.NormalizedTransfer{
		SourceOwnerAddress:      "walletA",
		DestinationOwnerAddress: "walletA",
	}
	if _, ok := MapWalletRelation("walletA", self); ok {
		t.Fatal("expected self-transfer to be excluded")
	}

	unresolved := transactions.NormalizedTransfer{
		SourceOwnerAddress:      "walletA",
		DestinationOwnerAddress: "",
	}
	if _, ok := MapWalletRelation("walletA", unresolved); ok {
		t.Fatal("expected unresolved owner endpoints to be excluded")
	}
}

func TestDeriveEventBuildsCounterpartyEvent(t *testing.T) {
	at := time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)
	tr := transactions.NormalizedTransfer{
		SourceOwnerAddress:      "walletB",
		DestinationOwnerAddress: "walletA",
		BlockTime:               at,
	}

	event, ok := DeriveEvent(42, "walletA", tr)
	if !ok {
		t.Fatal("expected derived event")
	}
	if event.FocalWalletID != 42 || event.RelationType != RelationReceiver || event.CounterpartyAddress != "walletB" {
		t.Fatalf("unexpected event: %#v", event)
	}
	if !event.OccurredAt.Equal(at) {
		t.Fatalf("unexpected occurred_at: %s", event.OccurredAt)
	}
}
