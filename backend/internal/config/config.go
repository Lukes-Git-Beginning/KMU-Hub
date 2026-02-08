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
	CRMGRPCPort     string `env:"CRM_GRPC_PORT,default=:50052"`
	CRMGRPCAddress  string `env:"CRM_GRPC_ADDRESS,default=localhost:50052"`
	ChatGRPCPort             string `env:"CHAT_GRPC_PORT,default=:50053"`
	ChatGRPCAddress          string `env:"CHAT_GRPC_ADDRESS,default=localhost:50053"`
	NotificationGRPCPort    string `env:"NOTIFICATION_GRPC_PORT,default=:50054"`
	NotificationGRPCAddress string `env:"NOTIFICATION_GRPC_ADDRESS,default=localhost:50054"`
	GatewayHTTPPort          string `env:"GATEWAY_HTTP_PORT,default=:8080"`

	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS,delimiter=;,default=http://localhost:3000;http://localhost:5173"`

	RateLimitRPS int `env:"RATE_LIMIT_RPS,default=100"`

	MetricsPort    string `env:"METRICS_PORT,default=:9090"`
	HealthPort     string `env:"HEALTH_PORT,default=:9091"`
	CRMHealthPort  string `env:"CRM_HEALTH_PORT,default=:9092"`
	ChatHealthPort         string `env:"CHAT_HEALTH_PORT,default=:9093"`
	NotificationHealthPort string `env:"NOTIFICATION_HEALTH_PORT,default=:9094"`

	// MinIO (S3-compatible file storage)
	MinIOEndpoint   string `env:"MINIO_ENDPOINT,default=localhost:9000"`
	MinIOAccessKey  string `env:"MINIO_ACCESS_KEY,default=kmuhub"`
	MinIOSecretKey  string `env:"MINIO_SECRET_KEY,default=kmuhub_dev"`
	MinIOBucket     string `env:"MINIO_BUCKET,default=kmuhub-files"`
	MinIOUseSSL     bool   `env:"MINIO_USE_SSL,default=false"`
	FileSizeLimitMB int    `env:"FILE_SIZE_LIMIT_MB,default=50"`
}

func Load(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
