package config_test

import (
	"os"
	"testing"

	"github.com/fagbenjaenoch/dorms-ng/internal/config"
)

func TestLoadConfig_Valid(t *testing.T) {
	// Set required environment variables
	os.Setenv("APP_PRIMARY.ENV", "development")
	os.Setenv("APP_SERVER.PORT", "8080")
	os.Setenv("APP_SERVER.READ_TIMEOUT", "10")
	os.Setenv("APP_SERVER.WRITE_TIMEOUT", "10")
	os.Setenv("APP_SERVER.IDLE_TIMEOUT", "30")
	os.Setenv("APP_SERVER.CORS_ALLOWED_ORIGINS", "*")
	os.Setenv("APP_LOGGING.FORMAT", "json")
	os.Setenv("APP_OBSERVABILITY.APP_NAME", "dorms-ng")
	os.Setenv("APP_OBSERVABILITY.ENVIRONMENT", "development")
	os.Setenv("APP_OBSERVABILITY.ENDPOINT", "http://localhost:4318/v1/traces")
	os.Setenv("APP_OBSERVABILITY.LOGGING_ENDPOINT", "http://localhost:4318/v1/logs")
	os.Setenv("APP_AUTH.JWT_SECRET", "test-secret")
	os.Setenv("APP_R2.ACCESS_KEY_ID", "test-key")
	os.Setenv("APP_R2.SECRET_ACCESS_KEY", "test-secret")
	os.Setenv("APP_R2.SECRET_KEY", "test-secret")
	os.Setenv("APP_R2.BUCKET", "test-bucket")
	os.Setenv("APP_R2.ENDPOINT", "https://test.r2.cloudflarestorage.com")
	os.Setenv("APP_DB.URI", "postgres://user:pass@localhost:5432/db")
	os.Setenv("APP_DB.NAME", "test-db")
	os.Setenv("APP_INFISICAL.CLIENT_ID", "infisical-client-id")
	os.Setenv("APP_INFISICAL.CLIENT_SECRET", "infisical-client-secret")
	os.Setenv("APP_INFISICAL.PROJECT_ID", "test-project-id")
	os.Setenv("APP_INFISICAL.NATS_SECRET_PATH", "/nats-secret-path")
	os.Setenv("APP_NATS.URL", "nats://localhost:4222")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Errorf("config could not be loaded, %s", err)
	}

	if cfg.Primary.Env != "development" {
		t.Errorf("expected 'development', got %q", cfg.Primary.Env)
	}

	if cfg.Server.Port != "8080" {
		t.Errorf("expected port 8080, got %s", cfg.Server.Port)
	}
}

func TestLoadConfig_InvalidEnv(t *testing.T) {
	os.Setenv("APP_PRIMARY.ENV", "invalid")
	os.Setenv("APP_SERVER.PORT", "hehehe")
	os.Setenv("APP_SERVER.READ_TIMEOUT", "hi")
	os.Setenv("APP_SERVER.WRITE_TIMEOUT", "ollo")
	os.Setenv("APP_SERVER.IDLE_TIMEOUT", "hello")
	os.Setenv("APP_SERVER.CORS_ALLOWED_ORIGINS", "*")
	os.Setenv("APP_LOGGING.FORMAT", "i_dont_care")
	os.Setenv("APP_OBSERVABILITY.APP_NAME", "2026")
	os.Setenv("APP_OBSERVABILITY.ENVIRONMENT", "invalid")
	os.Setenv("APP_OBSERVABILITY.ENDPOINT", "invalid")
	os.Setenv("APP_OBSERVABILITY.LOGGING_ENDPOINT", "invalid")
	os.Setenv("APP_AUTH.JWT_SECRET", "test-secret")
	os.Setenv("APP_R2.ACCESS_KEY_ID", "test-key")
	os.Setenv("APP_R2.SECRET_ACCESS_KEY", "test-secret")
	os.Setenv("APP_R2.SECRET_KEY", "test-secret")
	os.Setenv("APP_R2.BUCKET", "test-bucket")
	os.Setenv("APP_R2.ENDPOINT", "invalid")
	os.Setenv("APP_DB.URI", "invalid")
	_, err := config.LoadConfig()

	if err == nil {
		t.Error("should return error due to invalid config")
	}
}
