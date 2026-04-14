package counterparties

import "time"

type Counterparty struct {
	ID                  int64
	FocalWalletID       int64
	CounterpartyAddress string
	FirstSeenAt         time.Time
	LastSeenAt          time.Time
	InteractionCount    int64
	FirstInboundAt      *time.Time
	LastInboundAt       *time.Time
	InboundCount        int64
	FirstOutboundAt     *time.Time
	LastOutboundAt      *time.Time
	OutboundCount       int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
