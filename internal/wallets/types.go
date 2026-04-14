package wallets

import "time"

type Wallet struct {
	ID           int64
	Address      string
	Label        string
	Source       string
	CreatedAt    time.Time
	LastSyncedAt *time.Time
	Priority     int
}
