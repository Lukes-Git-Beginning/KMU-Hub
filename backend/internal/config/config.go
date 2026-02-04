package config

import (
	"context"
	"time"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	DatabaseURL string `env:"DATABASE_URL,default=postgres://kmuhub:kmuhub_dev@localhost:5432/kmuhub?sslmode=disable"`
	RedisURL    string `env:"REDIS_URL,default=redis://localhost:6379"`

	JWTSecret          string        `env:"JWT_SECRET,required"`
	AccessTokenExpiry  time.Duration `env:"ACCESS_TOKEN_EXPIRY,default=15m"`
	RefreshTokenExpiry time.Duration `env:"REFRESH_TOKEN_EXPIRY,default=168h"`

	AuthGRPCPort    string `env:"AUTH_GRPC_PORT,default=:50051"`
	AuthGRPCAddress string `env:"AUTH_GRPC_ADDRESS,default=localhost:50051"`
	GatewayHTTPPort string `env:"GATEWAY_HTTP_PORT,default=:8080"`

	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS,delimiter=;,default=http://localhost:3000"`

	RateLimitRPS int `env:"RATE_LIMIT_RPS,default=100"`

	MetricsPort string `env:"METRICS_PORT,default=:9090"`
	HealthPort  string `env:"HEALTH_PORT,default=:9091"`
}

func Load(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
