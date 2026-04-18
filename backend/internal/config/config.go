package config

import (
	"context"
	"fmt"
	"strings"
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
	BizGRPCPort             string `env:"BIZ_GRPC_PORT,default=:50058"`
	BizGRPCAddress          string `env:"BIZ_GRPC_ADDRESS,default=localhost:50058"`
	AutomationGRPCPort      string `env:"AUTOMATION_GRPC_PORT,default=:50059"`
	AutomationGRPCAddress   string `env:"AUTOMATION_GRPC_ADDRESS,default=localhost:50059"`
	PluginGRPCPort          string `env:"PLUGIN_GRPC_PORT,default=:50060"`
	PluginGRPCAddress       string `env:"PLUGIN_GRPC_ADDRESS,default=localhost:50060"`
	DialerGRPCPort          string `env:"DIALER_GRPC_PORT,default=:50061"`
	DialerGRPCAddress       string `env:"DIALER_GRPC_ADDRESS,default=localhost:50061"`
	WikiGRPCPort            string `env:"WIKI_GRPC_PORT,default=:50062"`
	WikiGRPCAddress         string `env:"WIKI_GRPC_ADDRESS,default=localhost:50062"`
	HelpdeskGRPCPort        string `env:"HELPDESK_GRPC_PORT,default=:50065"`
	HelpdeskGRPCAddress     string `env:"HELPDESK_GRPC_ADDRESS,default=localhost:50065"`
	GatewayHTTPPort          string `env:"GATEWAY_HTTP_PORT,default=:8080"`

	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS,delimiter=;,default=http://localhost:3000;http://localhost:5173"`

	RateLimitRPS int  `env:"RATE_LIMIT_RPS,default=100"`
	BehindProxy  bool `env:"BEHIND_PROXY,default=false"`

	MetricsPort    string `env:"METRICS_PORT,default=:9090"`
	HealthPort     string `env:"HEALTH_PORT,default=:9091"`
	CRMHealthPort  string `env:"CRM_HEALTH_PORT,default=:9092"`
	ChatHealthPort         string `env:"CHAT_HEALTH_PORT,default=:9093"`
	NotificationHealthPort string `env:"NOTIFICATION_HEALTH_PORT,default=:9094"`
	WorkHealthPort         string `env:"WORK_HEALTH_PORT,default=:9095"`
	EmailHealthPort        string `env:"EMAIL_HEALTH_PORT,default=:9096"`
	DocumentHealthPort     string `env:"DOCUMENT_HEALTH_PORT,default=:9097"`
	BizHealthPort          string `env:"BIZ_HEALTH_PORT,default=:9098"`
	AutomationHealthPort   string `env:"AUTOMATION_HEALTH_PORT,default=:9099"`
	PluginHealthPort       string `env:"PLUGIN_HEALTH_PORT,default=:9100"`
	DialerHealthPort       string `env:"DIALER_HEALTH_PORT,default=:9101"`
	WikiHealthPort         string `env:"WIKI_HEALTH_PORT,default=:9102"`
	HelpdeskHealthPort     string `env:"HELPDESK_HEALTH_PORT,default=:9105"`

	// LiveKit (Video calls -- optional, feature-flagged)
	LiveKitAPIKey    string `env:"LIVEKIT_API_KEY,default="`
	LiveKitAPISecret string `env:"LIVEKIT_API_SECRET,default="`
	LiveKitWSURL     string `env:"LIVEKIT_WS_URL,default="`

	// LiveKit Egress (Recording -- optional, requires LiveKit Egress service)
	LiveKitEgressTemplateURL string `env:"LIVEKIT_EGRESS_TEMPLATE_URL,default="`

	// LiveKit Webhook (optional, validates incoming LiveKit webhook signatures)
	LiveKitWebhookSecret string `env:"LIVEKIT_WEBHOOK_SECRET,default="`

	// TURN / coturn (self-hosted on separate Hetzner CPX11, optional — flag-off until provisioned)
	TURNSecret string `env:"TURN_SECRET,default="`
	COTURNHost string `env:"COTURN_HOST,default="`

	// Lexware Webhook HMAC (optional, validates incoming Lexware webhook signatures)
	LexwareWebhookSecret string `env:"LEXWARE_WEBHOOK_SECRET,default="`

	// Vault (secret encryption -- required for auth service)
	VaultMasterSecret string `env:"VAULT_MASTER_SECRET,default="`

	// WOPI (OnlyOffice collaborative editing)
	WOPIJWTSecret string `env:"WOPI_JWT_SECRET,default=wopi-dev-secret-change-me"`

	// CalDAV/CardDAV (external client sync)
	CalDAVEnabled bool `env:"CALDAV_ENABLED,default=false"`

	// Bexio Integration (optional)
	BexioClientID     string `env:"BEXIO_CLIENT_ID,default="`
	BexioClientSecret string `env:"BEXIO_CLIENT_SECRET,default="`
	BexioRedirectURL  string `env:"BEXIO_REDIRECT_URL,default="`

	// Lexware Office Integration (optional)
	LexwareAPIBaseURL string `env:"LEXWARE_API_BASE_URL,default=https://api.lexware.io"`

	// DATEV API Integration (optional)
	DatevClientID     string `env:"DATEV_CLIENT_ID,default="`
	DatevClientSecret string `env:"DATEV_CLIENT_SECRET,default="`
	DatevTokenURL     string `env:"DATEV_TOKEN_URL,default=https://login.datev.de/openidconnect/token"`
	DatevAuthURL      string `env:"DATEV_AUTH_URL,default=https://login.datev.de/openidconnect/authorize"`
	DatevAPIBaseURL   string `env:"DATEV_API_BASE_URL,default=https://accounting-documents.api.datev.de"`

	// gRPC mTLS (optional — if all three are set, service-to-service gRPC uses mTLS)
	GRPCTLSCertFile string `env:"GRPC_TLS_CERT_FILE,default="`
	GRPCTLSKeyFile  string `env:"GRPC_TLS_KEY_FILE,default="`
	GRPCTLSCAFile   string `env:"GRPC_TLS_CA_FILE,default="`

	// MinIO (S3-compatible file storage)
	MinIOEndpoint   string `env:"MINIO_ENDPOINT,default=localhost:9000"`
	MinIOAccessKey  string `env:"MINIO_ACCESS_KEY,default=kmuhub"`
	MinIOSecretKey  string `env:"MINIO_SECRET_KEY,default=kmuhub_dev"`
	MinIOBucket     string `env:"MINIO_BUCKET,default=kmuhub-files"`
	MinIOUseSSL     bool   `env:"MINIO_USE_SSL,default=false"`
	FileSizeLimitMB int    `env:"FILE_SIZE_LIMIT_MB,default=50"`

	// Env controls production secret validation ("production" enables hard checks)
	Env string `env:"COSMI_ENV,default=development"`
}

func Load(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, err
	}
	if strings.EqualFold(cfg.Env, "production") {
		if err := validateProductionSecrets(&cfg); err != nil {
			return nil, fmt.Errorf("production secret validation failed: %w", err)
		}
	}
	return &cfg, nil
}

// knownDevSecrets lists values that are safe in dev but must never reach production.
var knownDevSecrets = map[string]string{
	"WOPI_JWT_SECRET":   "wopi-dev-secret-change-me",
	"MINIO_SECRET_KEY":  "kmuhub_dev",
	"VAULT_MASTER_SECRET": "",
}

func validateProductionSecrets(cfg *Config) error {
	type check struct {
		name  string
		value string
		dev   string
	}
	checks := []check{
		{"WOPI_JWT_SECRET", cfg.WOPIJWTSecret, knownDevSecrets["WOPI_JWT_SECRET"]},
		{"MINIO_SECRET_KEY", cfg.MinIOSecretKey, knownDevSecrets["MINIO_SECRET_KEY"]},
		{"VAULT_MASTER_SECRET", cfg.VaultMasterSecret, knownDevSecrets["VAULT_MASTER_SECRET"]},
	}
	var errs []string
	for _, c := range checks {
		if c.value == c.dev {
			if c.dev == "" {
				errs = append(errs, fmt.Sprintf("%s must be set in production (currently empty)", c.name))
			} else {
				errs = append(errs, fmt.Sprintf("%s must not use the dev default value in production", c.name))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("insecure secrets detected:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}
