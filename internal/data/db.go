package data

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DBConfig struct {
	DSN          string
	MaxOpenConns int
	MinConns     int
	MaxIdleTime  time.Duration
}

func OpenDB(cfg DBConfig) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, err
	}
	config.MaxConns = int32(cfg.MaxOpenConns)
	config.MinConns = int32(cfg.MinConns)
	config.MaxConnIdleTime = cfg.MaxIdleTime
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	dbpool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	err = dbpool.Ping(ctx)
	if err != nil {
		dbpool.Close()
		return nil, err
	}
	return dbpool, nil
}
