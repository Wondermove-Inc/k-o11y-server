package config

import (
	"encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBatchConfig_LoadFromEnv(t *testing.T) {
	tests := []struct {
		name                  string
		envServiceMapEnabled  string
		envServiceMapInterval string
		expectedEnabled       bool
		expectedInterval      time.Duration
		expectError           bool
		errorContains         string
	}{
		{
			name:                  "Valid configuration",
			envServiceMapEnabled:  "true",
			envServiceMapInterval: "20s",
			expectedEnabled:       true,
			expectedInterval:      20 * time.Second,
			expectError:           false,
		},
		{
			name:                  "Disabled batch servicemap",
			envServiceMapEnabled:  "false",
			envServiceMapInterval: "20s",
			expectedEnabled:       false,
			expectedInterval:      20 * time.Second,
			expectError:           false,
		},
		{
			name:                  "Default values when env not set",
			envServiceMapEnabled:  "",
			envServiceMapInterval: "",
			expectedEnabled:       true,
			expectedInterval:      20 * time.Second,
			expectError:           false,
		},
		{
			name:                  "Custom interval",
			envServiceMapEnabled:  "true",
			envServiceMapInterval: "60s",
			expectedEnabled:       true,
			expectedInterval:      60 * time.Second,
			expectError:           false,
		},
		{
			name:                  "Invalid interval format",
			envServiceMapEnabled:  "true",
			envServiceMapInterval: "invalid",
			expectedEnabled:       true,
			expectedInterval:      0,
			expectError:           true,
			errorContains:         "invalid batch servicemap interval",
		},
		{
			name:                  "Zero interval",
			envServiceMapEnabled:  "true",
			envServiceMapInterval: "0s",
			expectedEnabled:       true,
			expectedInterval:      0,
			expectError:           true,
			errorContains:         "batch servicemap interval must be greater than 0",
		},
		{
			name:                  "Negative interval",
			envServiceMapEnabled:  "true",
			envServiceMapInterval: "-10s",
			expectedEnabled:       true,
			expectedInterval:      -10 * time.Second,
			expectError:           true,
			errorContains:         "batch servicemap interval must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange: Set required ClickHouse environment variables for test
			os.Setenv("CLICKHOUSE_HOST", "localhost")
			os.Setenv("CLICKHOUSE_PORT", "9000")
			os.Setenv("CLICKHOUSE_DATABASE", "test")

			// Arrange: Set batch environment variables
			if tt.envServiceMapEnabled != "" {
				os.Setenv("BATCH_SERVICEMAP_ENABLED", tt.envServiceMapEnabled)
			} else {
				os.Unsetenv("BATCH_SERVICEMAP_ENABLED")
			}

			if tt.envServiceMapInterval != "" {
				os.Setenv("BATCH_SERVICEMAP_INTERVAL", tt.envServiceMapInterval)
			} else {
				os.Unsetenv("BATCH_SERVICEMAP_INTERVAL")
			}

			// Reset global config to ensure clean state
			globalConfig = nil

			// Act: Load configuration
			cfg, err := LoadConfig()

			// Assert: Verify results
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				assert.Equal(t, tt.expectedEnabled, cfg.Batch.ServiceMapEnabled)
				assert.Equal(t, tt.expectedInterval, cfg.Batch.ServiceMapInterval)
			}

			// Cleanup
			os.Unsetenv("BATCH_SERVICEMAP_ENABLED")
			os.Unsetenv("BATCH_SERVICEMAP_INTERVAL")
			os.Unsetenv("CLICKHOUSE_HOST")
			os.Unsetenv("CLICKHOUSE_PORT")
			os.Unsetenv("CLICKHOUSE_DATABASE")
		})
	}
}

func TestBatchConfig_Validation(t *testing.T) {
	tests := []struct {
		name          string
		batchConfig   BatchConfig
		expectMissing []string
	}{
		{
			name: "Valid batch config",
			batchConfig: BatchConfig{
				ServiceMapEnabled:  true,
				ServiceMapInterval: 20 * time.Second,
			},
			expectMissing: []string{},
		},
		{
			name: "Invalid interval (zero)",
			batchConfig: BatchConfig{
				ServiceMapEnabled:  true,
				ServiceMapInterval: 0,
			},
			expectMissing: []string{"batch.servicemap_interval"},
		},
		{
			name: "Invalid interval (negative)",
			batchConfig: BatchConfig{
				ServiceMapEnabled:  true,
				ServiceMapInterval: -10 * time.Second,
			},
			expectMissing: []string{"batch.servicemap_interval"},
		},
		{
			name: "Disabled batch servicemap (no validation needed)",
			batchConfig: BatchConfig{
				ServiceMapEnabled:  false,
				ServiceMapInterval: 0,
			},
			expectMissing: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cfg := &Config{
				Batch: tt.batchConfig,
			}

			// Act
			missing := ValidateRequiredConfigs(cfg)

			// Assert
			if len(tt.expectMissing) == 0 {
				// No missing configs expected
				for _, m := range missing {
					assert.NotContains(t, m, "batch")
				}
			} else {
				for _, expected := range tt.expectMissing {
					assert.Contains(t, missing, expected)
				}
			}
		})
	}
}

func TestExtractTenantFromJWT(t *testing.T) {
	// Helper to build a JWT with given payload
	makeJWT := func(payload string) string {
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
		body := base64.RawURLEncoding.EncodeToString([]byte(payload))
		return header + "." + body + ".fakesignature"
	}

	tests := []struct {
		name      string
		token     string
		expected  string
		expectErr bool
		errMsg    string
	}{
		{
			name:     "should extract tenant_id from valid JWT",
			token:    makeJWT(`{"tenant_id":"0000000086","iss":"ko11y"}`),
			expected: "0000000086",
		},
		{
			name:      "should fail when JWT has invalid format (not 3 parts)",
			token:     "not-a-jwt",
			expectErr: true,
			errMsg:    "invalid JWT format",
		},
		{
			name:      "should fail when JWT has empty string",
			token:     "",
			expectErr: true,
			errMsg:    "invalid JWT format",
		},
		{
			name:      "should fail when payload has no tenant_id",
			token:     makeJWT(`{"iss":"ko11y","exp":1806054036}`),
			expectErr: true,
			errMsg:    "tenant_id not found",
		},
		{
			name:      "should fail when payload has empty tenant_id",
			token:     makeJWT(`{"tenant_id":"","iss":"ko11y"}`),
			expectErr: true,
			errMsg:    "tenant_id not found",
		},
		{
			name:      "should fail when payload is invalid JSON",
			token:     "header." + base64.RawURLEncoding.EncodeToString([]byte("not-json")) + ".sig",
			expectErr: true,
			errMsg:    "failed to parse JWT payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractTenantFromJWT(tt.token)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestUsageReporterConfig_JWTParsing(t *testing.T) {
	// Build a real-looking JWT
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"tenant_id":"0000000086","iss":"ko11y"}`))
	testJWT := header + "." + payload + ".fakesig"

	tests := []struct {
		name           string
		envs           map[string]string
		expectedTenant string
		expectErr      bool
		errMsg         string
	}{
		{
			name: "should auto-extract tenant_id from JWT when MGMT_TENANT_ID not set",
			envs: map[string]string{
				"USAGE_REPORTER_ENABLED": "true",
				"MGMT_LICENSE_KEY":       testJWT,
			},
			expectedTenant: "0000000086",
		},
		{
			name: "should prefer MGMT_TENANT_ID over JWT when both set",
			envs: map[string]string{
				"USAGE_REPORTER_ENABLED": "true",
				"MGMT_LICENSE_KEY":       testJWT,
				"MGMT_TENANT_ID":         "OVERRIDE_ID",
			},
			expectedTenant: "OVERRIDE_ID",
		},
		{
			name: "should use default MGMT_BASE_URL when not set",
			envs: map[string]string{
				"USAGE_REPORTER_ENABLED": "true",
				"MGMT_LICENSE_KEY":       testJWT,
			},
			expectedTenant: "0000000086",
		},
		{
			name: "should fail when license key is invalid JWT and no tenant_id",
			envs: map[string]string{
				"USAGE_REPORTER_ENABLED": "true",
				"MGMT_LICENSE_KEY":       "not-a-jwt",
			},
			expectErr: true,
			errMsg:    "failed to extract from license key JWT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all related env vars
			for _, key := range []string{
				"USAGE_REPORTER_ENABLED", "MGMT_BASE_URL", "MGMT_LICENSE_KEY",
				"MGMT_TENANT_ID", "USAGE_REPORTER_INTERVAL", "USAGE_REPORTER_HTTP_TIMEOUT",
				"USAGE_REPORTER_MAX_RETRIES",
			} {
				os.Unsetenv(key)
			}

			for k, v := range tt.envs {
				os.Setenv(k, v)
			}

			cfg, err := loadUsageReporterConfig()
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedTenant, cfg.TenantID)
				assert.Equal(t, "https://<YOUR_MGMT_PORTAL_URL>", cfg.MgmtBaseURL)
			}

			// Cleanup
			for k := range tt.envs {
				os.Unsetenv(k)
			}
		})
	}
}
