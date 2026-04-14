package counterparties

import "time"

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
