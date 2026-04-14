package counterparties

import (
	"strings"
	"time"

	"poisontrace/internal/transactions"
)

type RelationType string

const (
	RelationSender   RelationType = "sender"
	RelationReceiver RelationType = "receiver"
)

type Event struct {
	FocalWalletID       int64
	CounterpartyAddress string
	RelationType        RelationType
	OccurredAt          time.Time
}

func DeriveEvent(focalWalletID int64, focalWalletAddress string, tr transactions.NormalizedTransfer) (Event, bool) {
	mapping, ok := MapWalletRelation(focalWalletAddress, tr)
	if !ok {
		return Event{}, false
	}
	return Event{
		FocalWalletID:       focalWalletID,
		CounterpartyAddress: mapping.CounterpartyAddress,
		RelationType:        mapping.RelationType,
		OccurredAt:          tr.BlockTime.UTC(),
	}, true
}

type RelationMapping struct {
	RelationType        RelationType
	CounterpartyAddress string
}

// MapWalletRelation determines sender/receiver relation from normalized owner endpoints only.
func MapWalletRelation(focalWalletAddress string, tr transactions.NormalizedTransfer) (RelationMapping, bool) {
	focal := strings.TrimSpace(focalWalletAddress)
	src := strings.TrimSpace(tr.SourceOwnerAddress)
	dst := strings.TrimSpace(tr.DestinationOwnerAddress)
	if focal == "" || src == "" || dst == "" {
		return RelationMapping{}, false
	}

	// Owner-level self-transfers never form counterparties.
	if src == dst {
		return RelationMapping{}, false
	}
	if src == focal {
		return RelationMapping{
			RelationType:        RelationSender,
			CounterpartyAddress: dst,
		}, true
	}
	if dst == focal {
		return RelationMapping{
			RelationType:        RelationReceiver,
			CounterpartyAddress: src,
		}, true
	}
	return RelationMapping{}, false
}

func ApplyEvent(cp Counterparty, event Event) Counterparty {
	if cp.InteractionCount == 0 || event.OccurredAt.Before(cp.FirstSeenAt) {
		cp.FirstSeenAt = event.OccurredAt
	}
	if event.OccurredAt.After(cp.LastSeenAt) {
		cp.LastSeenAt = event.OccurredAt
	}
	cp.InteractionCount++

	if event.RelationType == RelationReceiver {
		cp.InboundCount++
		if cp.FirstInboundAt == nil || event.OccurredAt.Before(*cp.FirstInboundAt) {
			t := event.OccurredAt
			cp.FirstInboundAt = &t
		}
		if cp.LastInboundAt == nil || event.OccurredAt.After(*cp.LastInboundAt) {
			t := event.OccurredAt
			cp.LastInboundAt = &t
		}
	}

	if event.RelationType == RelationSender {
		cp.OutboundCount++
		if cp.FirstOutboundAt == nil || event.OccurredAt.Before(*cp.FirstOutboundAt) {
			t := event.OccurredAt
			cp.FirstOutboundAt = &t
		}
		if cp.LastOutboundAt == nil || event.OccurredAt.After(*cp.LastOutboundAt) {
			t := event.OccurredAt
			cp.LastOutboundAt = &t
		}
	}

	return cp
}
