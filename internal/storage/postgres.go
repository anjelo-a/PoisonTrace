package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type PostgresStore struct {
	DB *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{DB: db}
}

func (s *PostgresStore) Ping(ctx context.Context) error {
	if s.DB == nil {
		return fmt.Errorf("nil db")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return s.DB.PingContext(ctx)
}
