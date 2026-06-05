package config

import (
	"context"
	"testing"
)

// prodJWT satisfies the production minimum-length requirement (32+ chars).
const prodJWT = "strong-production-jwt-secret-minimum-32-chars"

// setStrongProdSecrets sets every secret the production assertion checks to a
// strong value. Individual tests then override exactly one var to isolate the
// failure reason under test.
func setStrongProdSecrets(t *testing.T) {
	t.Helper()
	t.Setenv("COSMI_ENV", "production")
	t.Setenv("JWT_SECRET", prodJWT)
	t.Setenv("WOPI_JWT_SECRET", "strong-wopi-secret")
	t.Setenv("MINIO_ACCESS_KEY", "strong-minio-access")
	t.Setenv("MINIO_SECRET_KEY", "strong-minio-secret")
	t.Setenv("VAULT_MASTER_SECRET", "strong-vault-secret")
}

func TestValidateProductionSecrets(t *testing.T) {
	t.Run("dev defaults pass in development mode", func(t *testing.T) {
		t.Setenv("COSMI_ENV", "development")
		t.Setenv("JWT_SECRET", "any-jwt-secret") // short + unchecked in dev
		cfg, err := Load(context.Background())
		if err != nil {
			t.Fatalf("Load() should not fail in development with dev defaults: %v", err)
		}
		if cfg.WOPIJWTSecret != "wopi-dev-secret-change-me" {
			t.Error("expected dev default for WOPI_JWT_SECRET")
		}
	})

	t.Run("all strong secrets pass in production", func(t *testing.T) {
		setStrongProdSecrets(t)
		_, err := Load(context.Background())
		if err != nil {
			t.Fatalf("Load() should succeed with all strong secrets: %v", err)
		}
	})

	t.Run("production mode is case-insensitive", func(t *testing.T) {
		t.Setenv("COSMI_ENV", "Production")
		// compose dev JWT is always refused in production — proves the
		// case-insensitive match arms the validation.
		t.Setenv("JWT_SECRET", "docker-dev-secret-minimum-32-characters")
		_, err := Load(context.Background())
		if err == nil {
			t.Fatal("Load() should fail for 'Production' (case-insensitive)")
		}
	})

	// --- requirements: config defaults refused only for services that need the group ---

	t.Run("required vault refuses go-default in production", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("VAULT_MASTER_SECRET", "")
		if _, err := Load(context.Background(), RequireVault); err == nil {
			t.Fatal("Load(RequireVault) should fail in production with empty VAULT_MASTER_SECRET")
		}
	})

	t.Run("lean service ignores vault go-default in production", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("VAULT_MASTER_SECRET", "")
		if _, err := Load(context.Background()); err != nil {
			t.Fatalf("Load() without RequireVault should ignore empty VAULT_MASTER_SECRET: %v", err)
		}
	})

	t.Run("required wopi refuses go-default in production", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("WOPI_JWT_SECRET", "wopi-dev-secret-change-me")
		if _, err := Load(context.Background(), RequireWOPI); err == nil {
			t.Fatal("Load(RequireWOPI) should fail in production with dev-default WOPI_JWT_SECRET")
		}
	})

	t.Run("lean service ignores wopi go-default in production", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("WOPI_JWT_SECRET", "wopi-dev-secret-change-me")
		if _, err := Load(context.Background()); err != nil {
			t.Fatalf("Load() without RequireWOPI should ignore the WOPI go-default: %v", err)
		}
	})

	t.Run("required minio refuses go-default in production", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("MINIO_SECRET_KEY", "kmuhub_dev")
		if _, err := Load(context.Background(), RequireMinIO); err == nil {
			t.Fatal("Load(RequireMinIO) should fail in production with dev MINIO_SECRET_KEY")
		}
	})

	t.Run("lean service ignores minio go-default in production", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("MINIO_SECRET_KEY", "kmuhub_dev")
		if _, err := Load(context.Background()); err != nil {
			t.Fatalf("Load() without RequireMinIO should ignore the MinIO go-default: %v", err)
		}
	})

	// --- compose dev values: refused for EVERY service, required or not ---

	t.Run("compose dev value rejected in production — JWT_SECRET", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("JWT_SECRET", "docker-dev-secret-minimum-32-characters")
		if _, err := Load(context.Background()); err == nil {
			t.Fatal("Load() should fail in production with compose dev JWT_SECRET")
		}
	})

	t.Run("compose dev value rejected in production — WOPI_JWT_SECRET", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("WOPI_JWT_SECRET", "docker-dev-wopi-secret-minimum-32-characters")
		if _, err := Load(context.Background()); err == nil {
			t.Fatal("Load() should fail in production with compose dev WOPI_JWT_SECRET even without RequireWOPI")
		}
	})

	t.Run("compose dev value rejected in production — VAULT_MASTER_SECRET", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("VAULT_MASTER_SECRET", "docker-dev-vault-secret-minimum-32-characters-long")
		if _, err := Load(context.Background()); err == nil {
			t.Fatal("Load() should fail in production with compose dev VAULT_MASTER_SECRET even without RequireVault")
		}
	})

	t.Run("minioadmin rejected in production — MINIO_ACCESS_KEY", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("MINIO_ACCESS_KEY", "minioadmin")
		if _, err := Load(context.Background()); err == nil {
			t.Fatal("Load() should fail in production with MINIO_ACCESS_KEY=minioadmin even without RequireMinIO")
		}
	})

	t.Run("minioadmin rejected in production — MINIO_SECRET_KEY", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("MINIO_SECRET_KEY", "minioadmin")
		if _, err := Load(context.Background()); err == nil {
			t.Fatal("Load() should fail in production with MINIO_SECRET_KEY=minioadmin even without RequireMinIO")
		}
	})

	t.Run("all strong secrets pass with all requirements", func(t *testing.T) {
		setStrongProdSecrets(t)
		if _, err := Load(context.Background(), RequireVault, RequireMinIO, RequireWOPI); err != nil {
			t.Fatalf("Load() with all requirements should succeed with strong secrets: %v", err)
		}
	})

	// --- JWT minimum length ---

	t.Run("short JWT_SECRET rejected in production", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("JWT_SECRET", "only-25-characters-long-x")
		if _, err := Load(context.Background()); err == nil {
			t.Fatal("Load() should fail in production with JWT_SECRET below 32 chars")
		}
	})

	// --- TURN symmetry ---

	t.Run("half-configured TURN rejected in production — host without secret", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("COTURN_HOST", "turn.example.com")
		t.Setenv("TURN_SECRET", "")
		if _, err := Load(context.Background()); err == nil {
			t.Fatal("Load() should fail in production with COTURN_HOST but no TURN_SECRET")
		}
	})

	t.Run("half-configured TURN rejected in production — secret without host", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("COTURN_HOST", "")
		t.Setenv("TURN_SECRET", "some-turn-secret")
		if _, err := Load(context.Background()); err == nil {
			t.Fatal("Load() should fail in production with TURN_SECRET but no COTURN_HOST")
		}
	})

	t.Run("symmetric TURN passes in production", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("COTURN_HOST", "turn.example.com")
		t.Setenv("TURN_SECRET", "some-turn-secret")
		if _, err := Load(context.Background()); err != nil {
			t.Fatalf("Load() should succeed with symmetric TURN config: %v", err)
		}
	})

	// --- LiveKit secret validation ---

	t.Run("prod with livekit dev key rejected", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("LIVEKIT_API_KEY", "devkey")
		t.Setenv("LIVEKIT_API_SECRET", "real-livekit-secret")
		t.Setenv("LIVEKIT_WEBHOOK_SECRET", "real-webhook-secret")
		if _, err := Load(context.Background()); err == nil {
			t.Fatal("Load() should fail in production with LIVEKIT_API_KEY=devkey")
		}
	})

	t.Run("prod with livekit dev secret rejected", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("LIVEKIT_API_KEY", "real-livekit-key")
		t.Setenv("LIVEKIT_API_SECRET", "devsecret")
		t.Setenv("LIVEKIT_WEBHOOK_SECRET", "real-webhook-secret")
		if _, err := Load(context.Background()); err == nil {
			t.Fatal("Load() should fail in production with LIVEKIT_API_SECRET=devsecret")
		}
	})

	t.Run("prod with empty livekit webhook secret rejected", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("LIVEKIT_API_KEY", "real-livekit-key")
		t.Setenv("LIVEKIT_API_SECRET", "real-livekit-secret")
		t.Setenv("LIVEKIT_WEBHOOK_SECRET", "")
		if _, err := Load(context.Background()); err == nil {
			t.Fatal("Load() should fail in production when LiveKit is on but webhook secret is empty")
		}
	})

	t.Run("prod with no livekit config passes", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("LIVEKIT_API_KEY", "")
		t.Setenv("LIVEKIT_API_SECRET", "")
		t.Setenv("LIVEKIT_WEBHOOK_SECRET", "")
		if _, err := Load(context.Background()); err != nil {
			t.Fatalf("Load() should succeed when LiveKit is not configured: %v", err)
		}
	})

	t.Run("prod with real livekit keys passes", func(t *testing.T) {
		setStrongProdSecrets(t)
		t.Setenv("LIVEKIT_API_KEY", "prod-livekit-api-key-abc123")
		t.Setenv("LIVEKIT_API_SECRET", "prod-livekit-api-secret-xyz789")
		t.Setenv("LIVEKIT_WEBHOOK_SECRET", "prod-livekit-webhook-secret-qrs456")
		if _, err := Load(context.Background()); err != nil {
			t.Fatalf("Load() should succeed with all real LiveKit keys: %v", err)
		}
	})

	t.Run("dev env with livekit dev keys passes", func(t *testing.T) {
		t.Setenv("COSMI_ENV", "development")
		t.Setenv("JWT_SECRET", "any-jwt-secret")
		t.Setenv("LIVEKIT_API_KEY", "devkey")
		t.Setenv("LIVEKIT_API_SECRET", "devsecret")
		// No check in dev — must succeed
		if _, err := Load(context.Background()); err != nil {
			t.Fatalf("Load() should succeed in development with livekit dev keys: %v", err)
		}
	})
}
