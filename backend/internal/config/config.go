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

	AuthGRPCPort            string `env:"AUTH_GRPC_PORT,default=:50051"`
	AuthGRPCAddress         string `env:"AUTH_GRPC_ADDRESS,default=localhost:50051"`
	CRMGRPCPort             string `env:"CRM_GRPC_PORT,default=:50052"`
	CRMGRPCAddress          string `env:"CRM_GRPC_ADDRESS,default=localhost:50052"`
	ChatGRPCPort            string `env:"CHAT_GRPC_PORT,default=:50053"`
	ChatGRPCAddress         string `env:"CHAT_GRPC_ADDRESS,default=localhost:50053"`
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
	BerichteGRPCPort        string `env:"BERICHTE_GRPC_PORT,default=:50063"`
	BerichteGRPCAddress     string `env:"BERICHTE_GRPC_ADDRESS,default=localhost:50063"`
	FormulareGRPCPort       string `env:"FORMULARE_GRPC_PORT,default=:50064"`
	FormulareGRPCAddress    string `env:"FORMULARE_GRPC_ADDRESS,default=localhost:50064"`
	HelpdeskGRPCPort        string `env:"HELPDESK_GRPC_PORT,default=:50065"`
	HelpdeskGRPCAddress     string `env:"HELPDESK_GRPC_ADDRESS,default=localhost:50065"`
	InventarGRPCPort        string `env:"INVENTAR_GRPC_PORT,default=:50070"`
	InventarGRPCAddress     string `env:"INVENTAR_GRPC_ADDRESS,default=localhost:50070"`
	EinkaufGRPCPort         string `env:"EINKAUF_GRPC_PORT,default=:50071"`
	EinkaufGRPCAddress      string `env:"EINKAUF_GRPC_ADDRESS,default=localhost:50071"`
	ProduktionGRPCPort      string `env:"PRODUKTION_GRPC_PORT,default=:50072"`
	ProduktionGRPCAddress   string `env:"PRODUKTION_GRPC_ADDRESS,default=localhost:50072"`
	VertraegeGRPCPort       string `env:"VERTRAEGE_GRPC_PORT,default=:50073"`
	VertraegeGRPCAddress    string `env:"VERTRAEGE_GRPC_ADDRESS,default=localhost:50073"`
	RapporteGRPCPort        string `env:"RAPPORTE_GRPC_PORT,default=:50074"`
	RapporteGRPCAddress     string `env:"RAPPORTE_GRPC_ADDRESS,default=localhost:50074"`
	SchichtenGRPCPort       string `env:"SCHICHTEN_GRPC_PORT,default=:50075"`
	SchichtenGRPCAddress    string `env:"SCHICHTEN_GRPC_ADDRESS,default=localhost:50075"`
	FuhrparkGRPCPort        string `env:"FUHRPARK_GRPC_PORT,default=:50076"`
	FuhrparkGRPCAddress     string `env:"FUHRPARK_GRPC_ADDRESS,default=localhost:50076"`
	VermietungGRPCPort      string `env:"VERMIETUNG_GRPC_PORT,default=:50077"`
	VermietungGRPCAddress   string `env:"VERMIETUNG_GRPC_ADDRESS,default=localhost:50077"`
	GatewayHTTPPort         string `env:"GATEWAY_HTTP_PORT,default=:8080"`

	CORSAllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS,delimiter=;,default=http://localhost:3000;http://localhost:5173"`

	RateLimitRPS int  `env:"RATE_LIMIT_RPS,default=100"`
	BehindProxy  bool `env:"BEHIND_PROXY,default=false"`

	MetricsPort            string `env:"METRICS_PORT,default=:9090"`
	HealthPort             string `env:"HEALTH_PORT,default=:9091"`
	CRMHealthPort          string `env:"CRM_HEALTH_PORT,default=:9092"`
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
	BerichteHealthPort     string `env:"BERICHTE_HEALTH_PORT,default=:9103"`
	FormulareHealthPort    string `env:"FORMULARE_HEALTH_PORT,default=:9104"`
	HelpdeskHealthPort     string `env:"HELPDESK_HEALTH_PORT,default=:9105"`
	InventarHealthPort     string `env:"INVENTAR_HEALTH_PORT,default=:9110"`
	EinkaufHealthPort      string `env:"EINKAUF_HEALTH_PORT,default=:9111"`
	ProduktionHealthPort   string `env:"PRODUKTION_HEALTH_PORT,default=:9112"`
	VertraegeHealthPort    string `env:"VERTRAEGE_HEALTH_PORT,default=:9113"`
	RapporteHealthPort     string `env:"RAPPORTE_HEALTH_PORT,default=:9114"`
	SchichtenHealthPort    string `env:"SCHICHTEN_HEALTH_PORT,default=:9115"`
	FuhrparkHealthPort     string `env:"FUHRPARK_HEALTH_PORT,default=:9116"`
	VermietungHealthPort   string `env:"VERMIETUNG_HEALTH_PORT,default=:9117"`

	// LiveKit (Video calls -- optional, feature-flagged)
	LiveKitAPIKey    string `env:"LIVEKIT_API_KEY,default="`
	LiveKitAPISecret string `env:"LIVEKIT_API_SECRET,default="`
	LiveKitWSURL     string `env:"LIVEKIT_WS_URL,default="`
	// LiveKit server API endpoint for room/egress management (twirp over HTTP).
	// Inside docker the public WSS URL is not routable from the backend, so this
	// points at the internal service address (ws://livekit:7880). Empty falls
	// back to LIVEKIT_WS_URL — see LiveKitServerAPIURL().
	LiveKitInternalURL string `env:"LIVEKIT_INTERNAL_URL,default="`

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

// LiveKitServerAPIURL returns the URL backend services use for LiveKit server
// API calls (room create/delete, egress — twirp over HTTP). Prefers the
// internal docker-network address; falls back to the public WS URL so
// single-host setups without the override keep working.
func (c *Config) LiveKitServerAPIURL() string {
	if c.LiveKitInternalURL != "" {
		return c.LiveKitInternalURL
	}
	return c.LiveKitWSURL
}

// Load reads the configuration from the environment. Services pass the secret
// groups they actually consume (RequireVault, RequireMinIO, RequireWOPI) so the
// production assertion can demand real values for them; lean services call
// Load(ctx) and are only held to the universal checks (JWT, LiveKit-if-on, TURN).
func Load(ctx context.Context, requirements ...Requirement) (*Config, error) {
	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, err
	}
	if strings.EqualFold(cfg.Env, "production") {
		if err := validateProductionSecrets(&cfg, requirements); err != nil {
			return nil, fmt.Errorf("production secret validation failed: %w", err)
		}
	}
	return &cfg, nil
}

// Requirement marks a secret group a service actually consumes. Production
// validation enforces presence + strength only for required groups — a lean
// module service (formulare, vertraege, …) never receives WOPI/MinIO/Vault env
// from compose and must not be refused over their unused Go defaults.
// Explicitly configured dev values are rejected regardless of requirements.
type Requirement int

const (
	// RequireVault — service decrypts/encrypts via VAULT_MASTER_SECRET (auth, email, biz).
	RequireVault Requirement = iota
	// RequireMinIO — service talks to MinIO (chat, work, email, document, gateway).
	RequireMinIO
	// RequireWOPI — service signs/validates WOPI tokens (document, gateway).
	RequireWOPI
)

// composeDevSecrets are the docker-compose dev values. A service that receives
// one of these in production got it through a compose/overlay gap — always refuse,
// whether or not the service declared the requirement.
var composeDevSecrets = map[string][]string{
	"JWT_SECRET":          {"docker-dev-secret-minimum-32-characters"},
	"WOPI_JWT_SECRET":     {"docker-dev-wopi-secret-minimum-32-characters"},
	"MINIO_ACCESS_KEY":    {"minioadmin"},
	"MINIO_SECRET_KEY":    {"minioadmin"},
	"VAULT_MASTER_SECRET": {"docker-dev-vault-secret-minimum-32-characters-long"},
	"LIVEKIT_API_KEY":     {"devkey"},
	"LIVEKIT_API_SECRET":  {"devsecret"},
}

// configDefaultSecrets are the Go-config defaults — they mean "env not set".
// Only refused when the service declared it requires the group.
var configDefaultSecrets = map[string][]string{
	"WOPI_JWT_SECRET":        {"wopi-dev-secret-change-me", ""},
	"MINIO_ACCESS_KEY":       {"kmuhub", ""},
	"MINIO_SECRET_KEY":       {"kmuhub_dev", ""},
	"VAULT_MASTER_SECRET":    {""},
	"LIVEKIT_WEBHOOK_SECRET": {""},
}

// minJWTSecretLength is enforced in production only — short HMAC keys make
// token forgery brute-forceable regardless of whether the value is a known default.
const minJWTSecretLength = 32

func validateProductionSecrets(cfg *Config, requirements []Requirement) error {
	required := make(map[Requirement]bool, len(requirements))
	for _, r := range requirements {
		required[r] = true
	}

	type check struct {
		name     string
		value    string
		required bool // also refuse the "env not set" config defaults
	}
	checks := []check{
		{"JWT_SECRET", cfg.JWTSecret, true},
		{"WOPI_JWT_SECRET", cfg.WOPIJWTSecret, required[RequireWOPI]},
		{"MINIO_ACCESS_KEY", cfg.MinIOAccessKey, required[RequireMinIO]},
		{"MINIO_SECRET_KEY", cfg.MinIOSecretKey, required[RequireMinIO]},
		{"VAULT_MASTER_SECRET", cfg.VaultMasterSecret, required[RequireVault]},
	}

	// LiveKit secrets are only validated when LiveKit is configured.
	// If both API key and secret are empty, LiveKit is OFF — no check needed.
	liveKitConfigured := cfg.LiveKitAPIKey != "" || cfg.LiveKitAPISecret != ""
	if liveKitConfigured {
		checks = append(checks,
			check{"LIVEKIT_API_KEY", cfg.LiveKitAPIKey, true},
			check{"LIVEKIT_API_SECRET", cfg.LiveKitAPISecret, true},
			// LIVEKIT_WEBHOOK_SECRET must be set when LiveKit is active in production.
			check{"LIVEKIT_WEBHOOK_SECRET", cfg.LiveKitWebhookSecret, true},
		)
	}

	var errs []string

	if len(cfg.JWTSecret) < minJWTSecretLength {
		errs = append(errs, fmt.Sprintf("JWT_SECRET must be at least %d characters in production", minJWTSecretLength))
	}

	// TURN must be configured symmetrically. A half-configured TURN (host without
	// secret, or vice-versa) issues join tokens with TURN URLs but invalid credentials,
	// so the client probes TURN, gets HTTP 401 from coturn, and falls back to STUN
	// after a noticeable latency spike. Refuse to start in production when only one
	// of the two values is set.
	turnHostSet := cfg.COTURNHost != ""
	turnSecretSet := cfg.TURNSecret != ""
	if turnHostSet != turnSecretSet {
		if turnHostSet {
			errs = append(errs, "COTURN_HOST is set but TURN_SECRET is empty — half-configured TURN issues invalid credentials")
		} else {
			errs = append(errs, "TURN_SECRET is set but COTURN_HOST is empty — secret has no host to authenticate against")
		}
	}

	for _, c := range checks {
		denied := composeDevSecrets[c.name]
		if c.required {
			denied = append(denied, configDefaultSecrets[c.name]...)
		}
		for _, dev := range denied {
			if c.value == dev {
				if dev == "" {
					errs = append(errs, fmt.Sprintf("%s must be set in production (currently empty)", c.name))
				} else {
					errs = append(errs, fmt.Sprintf("%s must not use the dev default value in production", c.name))
				}
				break
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("insecure secrets detected:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}
