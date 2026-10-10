package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceIdentifier(t *testing.T) {
	assert.Equal(t, "https://mcp.example.com/mcp", ResourceIdentifier("https://mcp.example.com"))
	assert.Equal(t, "https://mcp.example.com/mcp", ResourceIdentifier("https://mcp.example.com/"))
}

// The protected resource metadata names the MCP endpoint URL as the resource
// at the root well-known path and at the path-aware one, so a client that
// follows the 401 of the endpoint (RFC 9728 §3.3) and one that derives the
// indicator from the endpoint URL ask for the same audience.
func TestProtectedResourceMetadata_NamesTheMCPEndpoint(t *testing.T) {
	handler, err := NewHandler(&Config{
		BaseURL:            "http://localhost:8080",
		GoogleClientID:     "test-client-id",
		GoogleClientSecret: "test-client-secret",
	})
	require.NoError(t, err)
	defer handler.Stop()

	mux := http.NewServeMux()
	handler.GetHandler().RegisterProtectedResourceMetadataRoutes(mux, MCPEndpointPath)

	for _, path := range []string{
		"/.well-known/oauth-protected-resource",
		"/.well-known/oauth-protected-resource/mcp",
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, rec.Code, path)

		var metadata struct {
			Resource             string   `json:"resource"`
			AuthorizationServers []string `json:"authorization_servers"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &metadata), path)
		assert.Equal(t, "http://localhost:8080/mcp", metadata.Resource, path)
		assert.Equal(t, []string{"http://localhost:8080"}, metadata.AuthorizationServers, path)
	}
}
