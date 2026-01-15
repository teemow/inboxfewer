package oauth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHandler(t *testing.T) {
	config := &Config{
		BaseURL:            "http://localhost:8080",
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-client-secret",
		Security: SecurityConfig{
			AllowPublicClientRegistration: false,
			RegistrationAccessToken:       "test-token",
			MaxClientsPerIP:               10,
			EnableAuditLogging:            true,
			RefreshTokenTTL:               90 * 24 * time.Hour,
		},
		RateLimit: RateLimitConfig{
			Rate:      10,
			Burst:     20,
			UserRate:  100,
			UserBurst: 200,
		},
	}

	handler, err := NewHandler(config)
	require.NoError(t, err)
	require.NotNil(t, handler)
	defer handler.Stop()

	// Verify handler components are initialized
	assert.NotNil(t, handler.GetHandler())
	assert.NotNil(t, handler.GetStore())
	assert.NotNil(t, handler.GetServer())

	// Verify token refresh capability
	assert.True(t, handler.CanRefreshTokens())
}

func TestNewHandler_MinimalConfig(t *testing.T) {
	// Test with minimal configuration (using defaults)
	config := &Config{
		BaseURL:            "http://localhost:8080",
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-client-secret",
	}

	handler, err := NewHandler(config)
	require.NoError(t, err)
	require.NotNil(t, handler)
	defer handler.Stop()

	assert.NotNil(t, handler.GetHandler())
	assert.NotNil(t, handler.GetStore())
}

func TestNewHandler_WithEncryption(t *testing.T) {
	// Create a 32-byte encryption key
	encryptionKey := make([]byte, 32)
	for i := range encryptionKey {
		encryptionKey[i] = byte(i)
	}

	config := &Config{
		BaseURL:            "http://localhost:8080",
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-client-secret",
		Security: SecurityConfig{
			EncryptionKey:      encryptionKey,
			EnableAuditLogging: true,
		},
	}

	handler, err := NewHandler(config)
	require.NoError(t, err)
	require.NotNil(t, handler)
	defer handler.Stop()

	assert.NotNil(t, handler.GetHandler())
}

func TestHandler_Stop(t *testing.T) {
	config := &Config{
		BaseURL:            "http://localhost:8080",
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-client-secret",
		RateLimit: RateLimitConfig{
			Rate:     10,
			UserRate: 100,
		},
	}

	handler, err := NewHandler(config)
	require.NoError(t, err)
	require.NotNil(t, handler)

	// Stop should not panic
	assert.NotPanics(t, func() {
		handler.Stop()
	})

	// Calling Stop multiple times should be safe
	assert.NotPanics(t, func() {
		handler.Stop()
	})
}

func TestNewHandler_WithCIMDOptions(t *testing.T) {
	// Test CIMD options are passed to the mcp-oauth server configuration
	tests := []struct {
		name                string
		enableCIMD          bool
		cimdAllowPrivateIPs bool
	}{
		{
			name:                "CIMD enabled, private IPs blocked",
			enableCIMD:          true,
			cimdAllowPrivateIPs: false,
		},
		{
			name:                "CIMD enabled, private IPs allowed",
			enableCIMD:          true,
			cimdAllowPrivateIPs: true,
		},
		{
			name:                "CIMD disabled",
			enableCIMD:          false,
			cimdAllowPrivateIPs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				BaseURL:             "http://localhost:8080",
				GoogleClientID:      "test-client-id",
				GoogleClientSecret:  "test-client-secret",
				EnableCIMD:          tt.enableCIMD,
				CIMDAllowPrivateIPs: tt.cimdAllowPrivateIPs,
			}

			handler, err := NewHandler(config)
			require.NoError(t, err)
			require.NotNil(t, handler)
			defer handler.Stop()

			// Verify handler was created with the configuration
			assert.NotNil(t, handler.GetHandler())
			assert.NotNil(t, handler.GetServer())

			// Verify the server configuration
			serverConfig := handler.GetServer().Config
			assert.Equal(t, tt.enableCIMD, serverConfig.EnableClientIDMetadataDocuments)
			assert.Equal(t, tt.cimdAllowPrivateIPs, serverConfig.AllowPrivateIPClientMetadata)
		})
	}
}
