package health

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Status represents the overall health status.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// CheckResult is the result of a single health check.
type CheckResult struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Checker performs a health check on a dependency.
type Checker interface {
	Check(ctx context.Context) CheckResult
}

// CheckAll runs all checkers and returns the aggregate status.
// If any checker returns unhealthy, the overall status is unhealthy.
// If any checker returns degraded (and none unhealthy), overall is degraded.
func CheckAll(ctx context.Context, checkers []Checker) (Status, []CheckResult) {
	results := make([]CheckResult, 0, len(checkers))
	overall := StatusHealthy

	for _, c := range checkers {
		result := c.Check(ctx)
		results = append(results, result)

		switch result.Status {
		case StatusUnhealthy:
			overall = StatusUnhealthy
		case StatusDegraded:
			if overall != StatusUnhealthy {
				overall = StatusDegraded
			}
		}
	}

	return overall, results
}

// PostgresChecker checks PostgreSQL connectivity.
type PostgresChecker struct {
	pool *pgxpool.Pool
}

// NewPostgresChecker creates a PostgresChecker.
func NewPostgresChecker(pool *pgxpool.Pool) *PostgresChecker {
	return &PostgresChecker{pool: pool}
}

func (c *PostgresChecker) Check(ctx context.Context) CheckResult {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := c.pool.Ping(ctx); err != nil {
		return CheckResult{Name: "postgres", Status: StatusUnhealthy, Error: err.Error()}
	}
	return CheckResult{Name: "postgres", Status: StatusHealthy}
}

// RedisChecker checks Redis connectivity. Redis is best-effort,
// so a failure results in degraded rather than unhealthy.
type RedisChecker struct {
	client *redis.Client
}

// NewRedisChecker creates a RedisChecker.
func NewRedisChecker(client *redis.Client) *RedisChecker {
	return &RedisChecker{client: client}
}

func (c *RedisChecker) Check(ctx context.Context) CheckResult {
	if c.client == nil {
		return CheckResult{Name: "redis", Status: StatusDegraded, Error: "not configured"}
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := c.client.Ping(ctx).Err(); err != nil {
		return CheckResult{Name: "redis", Status: StatusDegraded, Error: err.Error()}
	}
	return CheckResult{Name: "redis", Status: StatusHealthy}
}
