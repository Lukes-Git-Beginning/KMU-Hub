package config

import (
	"context"
	"testing"
)

func TestValidateProductionSecrets(t *testing.T) {
	t.Run("dev defaults pass in development mode", func(t *testing.T) {
		t.Setenv("COSMI_ENV", "development")
		t.Setenv("JWT_SECRET", "any-jwt-secret")
		cfg, err := Load(context.Background())
		if err != nil {
			t.Fatalf("Load() should not fail in development with dev defaults: %v", err)
		}
		if cfg.WOPIJWTSecret != "wopi-dev-secret-change-me" {
			t.Error("expected dev default for WOPI_JWT_SECRET")
		}
	})

	t.Run("dev defaults rejected in production — WOPI_JWT_SECRET", func(t *testing.T) {
		t.Setenv("COSMI_ENV", "production")
		t.Setenv("JWT_SECRET", "any-jwt-secret")
		t.Setenv("MINIO_SECRET_KEY", "strong-minio-secret")
		t.Setenv("VAULT_MASTER_SECRET", "strong-vault-secret")
		// WOPI_JWT_SECRET not set → uses dev default
		_, err := Load(context.Background())
		if err == nil {
			t.Fatal("Load() should fail in production with dev WOPI_JWT_SECRET")
		}
	})

	t.Run("dev defaults rejected in production — MINIO_SECRET_KEY", func(t *testing.T) {
		t.Setenv("COSMI_ENV", "production")
		t.Setenv("JWT_SECRET", "any-jwt-secret")
		t.Setenv("WOPI_JWT_SECRET", "strong-wopi-secret")
		t.Setenv("VAULT_MASTER_SECRET", "strong-vault-secret")
		// MINIO_SECRET_KEY not set → uses dev default "kmuhub_dev"
		_, err := Load(context.Background())
		if err == nil {
			t.Fatal("Load() should fail in production with dev MINIO_SECRET_KEY")
		}
	})

	t.Run("empty VAULT_MASTER_SECRET rejected in production", func(t *testing.T) {
		t.Setenv("COSMI_ENV", "production")
		t.Setenv("JWT_SECRET", "any-jwt-secret")
		t.Setenv("WOPI_JWT_SECRET", "strong-wopi-secret")
		t.Setenv("MINIO_SECRET_KEY", "strong-minio-secret")
		t.Setenv("VAULT_MASTER_SECRET", "")
		_, err := Load(context.Background())
		if err == nil {
			t.Fatal("Load() should fail in production with empty VAULT_MASTER_SECRET")
		}
	})

	t.Run("all strong secrets pass in production", func(t *testing.T) {
		t.Setenv("COSMI_ENV", "production")
		t.Setenv("JWT_SECRET", "any-jwt-secret")
		t.Setenv("WOPI_JWT_SECRET", "super-secret-wopi-key-for-prod")
		t.Setenv("MINIO_SECRET_KEY", "super-secret-minio-key-for-prod")
		t.Setenv("VAULT_MASTER_SECRET", "super-secret-vault-key-for-prod")
		_, err := Load(context.Background())
		if err != nil {
			t.Fatalf("Load() should succeed with all strong secrets: %v", err)
		}
	})

	t.Run("production mode is case-insensitive", func(t *testing.T) {
		t.Setenv("COSMI_ENV", "Production")
		t.Setenv("JWT_SECRET", "any-jwt-secret")
		// dev defaults → should still fail
		_, err := Load(context.Background())
		if err == nil {
			t.Fatal("Load() should fail for 'Production' (case-insensitive)")
		}
	})
}
