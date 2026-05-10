package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = time.Minute
	config.ConnConfig.ConnectTimeout = 10 * time.Second

	// AfterRelease clears the RLS session GUCs before the connection
	// returns to the pool. BeginRLSTx already sets them with LOCAL scope
	// (revert at COMMIT/ROLLBACK), so this is defence-in-depth against
	// any code path that runs SQL outside a managed transaction. Returning
	// false discards the connection on reset failure rather than handing
	// back a possibly-tainted one.
	config.AfterRelease = func(conn *pgx.Conn) bool {
		_, err := conn.Exec(context.Background(),
			`SELECT set_config('app.tenant_id', '', false),
                    set_config('app.user_id',   '', false),
                    set_config('app.role',      '', false)`)
		return err == nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
