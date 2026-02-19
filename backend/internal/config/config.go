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
	WorkGRPCPort            string `env:"WORK_GRPC_PORT,default=:50055"`
	WorkGRPCAddress         string `env:"WORK_GRPC_ADDRESS,default=localhost:50055"`
	EmailGRPCPort           string `env:"EMAIL_GRPC_PORT,default=:50056"`
	EmailGRPCAddress        string `env:"EMAIL_GRPC_ADDRESS,default=localhost:50056"`
	DocumentGRPCPort        string `env:"DOCUMENT_GRPC_PORT,default=:50057"`
	DocumentGRPCAddress     string `env:"DOCUMENT_GRPC_ADDRESS,default=localhost:50057"`
	GatewayHTTPPort          string `env:"GATEWAY_HTTP_PORT,default=:8080"`

	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS,delimiter=;,default=http://localhost:3000;http://localhost:5173"`

	RateLimitRPS int `env:"RATE_LIMIT_RPS,default=100"`

	MetricsPort    string `env:"METRICS_PORT,default=:9090"`
	HealthPort     string `env:"HEALTH_PORT,default=:9091"`
	CRMHealthPort  string `env:"CRM_HEALTH_PORT,default=:9092"`
	ChatHealthPort         string `env:"CHAT_HEALTH_PORT,default=:9093"`
	NotificationHealthPort string `env:"NOTIFICATION_HEALTH_PORT,default=:9094"`
	WorkHealthPort         string `env:"WORK_HEALTH_PORT,default=:9095"`
	EmailHealthPort        string `env:"EMAIL_HEALTH_PORT,default=:9096"`
	DocumentHealthPort     string `env:"DOCUMENT_HEALTH_PORT,default=:9097"`

	// LiveKit (Video calls -- optional, feature-flagged)
	LiveKitAPIKey    string `env:"LIVEKIT_API_KEY,default="`
	LiveKitAPISecret string `env:"LIVEKIT_API_SECRET,default="`
	LiveKitWSURL     string `env:"LIVEKIT_WS_URL,default="`

	// LiveKit Egress (Recording -- optional, requires LiveKit Egress service)
	LiveKitEgressTemplateURL string `env:"LIVEKIT_EGRESS_TEMPLATE_URL,default="`

	// LiveKit Webhook (optional, validates incoming LiveKit webhook signatures)
	LiveKitWebhookSecret string `env:"LIVEKIT_WEBHOOK_SECRET,default="`

	// Vault (secret encryption -- required for auth service)
	VaultMasterSecret string `env:"VAULT_MASTER_SECRET,default="`

	// WOPI (OnlyOffice collaborative editing)
	WOPIJWTSecret string `env:"WOPI_JWT_SECRET,default=wopi-dev-secret-change-me"`

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
