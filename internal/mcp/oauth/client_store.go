package oauth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ClientStore manages registered OAuth clients
type ClientStore struct {
	clients       map[string]*RegisteredClient
	clientsPerIP  map[string]int // Track number of clients per IP for DoS protection
	mu            sync.RWMutex
	logger        *slog.Logger
}

// NewClientStore creates a new client store
func NewClientStore(logger *slog.Logger) *ClientStore {
	if logger == nil {
		logger = slog.Default()
	}

	return &ClientStore{
		clients:      make(map[string]*RegisteredClient),
		clientsPerIP: make(map[string]int),
		logger:       logger,
	}
}

// CheckIPLimit checks if an IP has reached the client registration limit
// Returns an error if the limit is reached
func (s *ClientStore) CheckIPLimit(ip string, maxClientsPerIP int) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if maxClientsPerIP <= 0 {
		return nil // No limit
	}

	count := s.clientsPerIP[ip]
	if count >= maxClientsPerIP {
		return fmt.Errorf("client registration limit reached for IP %s (%d/%d)", ip, count, maxClientsPerIP)
	}

	return nil
}

// RegisterClient registers a new OAuth client and returns the client info
// clientIP is used for DoS protection via per-IP limits
func (s *ClientStore) RegisterClient(req *ClientRegistrationRequest, clientIP string) (*ClientRegistrationResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate client ID
	clientID, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client ID: %w", err)
	}

	// Generate client secret
	clientSecret, err := generateSecureToken(48)
	if err != nil {
		return nil, fmt.Errorf("failed to generate client secret: %w", err)
	}

	// Hash the client secret for storage
	secretHash, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash client secret: %w", err)
	}

	now := time.Now().Unix()

	// Set defaults for missing fields
	tokenEndpointAuthMethod := req.TokenEndpointAuthMethod
	if tokenEndpointAuthMethod == "" {
		tokenEndpointAuthMethod = "client_secret_basic"
	}

	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code", "refresh_token"}
	}

	responseTypes := req.ResponseTypes
	if len(responseTypes) == 0 {
		responseTypes = []string{"code"}
	}

	// Create registered client
	client := &RegisteredClient{
		ClientID:                clientID,
		ClientSecret:            "", // Don't store plain text
		ClientSecretHash:        string(secretHash),
		ClientIDIssuedAt:        now,
		ClientSecretExpiresAt:   0, // Never expires
		RedirectURIs:            req.RedirectURIs,
		TokenEndpointAuthMethod: tokenEndpointAuthMethod,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		ClientName:              req.ClientName,
		Scope:                   req.Scope,
	}

	// Store the client
	s.clients[clientID] = client

	// Increment IP counter for DoS protection
	if clientIP != "" {
		s.clientsPerIP[clientIP]++
	}

	s.logger.Info("Registered new OAuth client",
		"client_id", clientID,
		"client_name", req.ClientName,
		"client_ip", clientIP,
		"clients_from_ip", s.clientsPerIP[clientIP],
		"redirect_uris", req.RedirectURIs,
		"grant_types", grantTypes,
	)

	// Return registration response
	return &ClientRegistrationResponse{
		ClientID:                clientID,
		ClientSecret:            clientSecret, // Only returned once
		ClientIDIssuedAt:        now,
		ClientSecretExpiresAt:   0,
		RedirectURIs:            req.RedirectURIs,
		TokenEndpointAuthMethod: tokenEndpointAuthMethod,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		ClientName:              req.ClientName,
		Scope:                   req.Scope,
	}, nil
}

// GetClient retrieves a registered client by ID
func (s *ClientStore) GetClient(clientID string) (*RegisteredClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, exists := s.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("client not found")
	}

	return client, nil
}

// ValidateClientSecret validates a client's secret
func (s *ClientStore) ValidateClientSecret(clientID, clientSecret string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, exists := s.clients[clientID]
	if !exists {
		return fmt.Errorf("client not found")
	}

	// Compare with bcrypt hash
	if err := bcrypt.CompareHashAndPassword([]byte(client.ClientSecretHash), []byte(clientSecret)); err != nil {
		return fmt.Errorf("invalid client secret")
	}

	return nil
}

// ValidateRedirectURI checks if a redirect URI is registered for a client
func (s *ClientStore) ValidateRedirectURI(clientID, redirectURI string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, exists := s.clients[clientID]
	if !exists {
		return fmt.Errorf("client not found")
	}

	// Check if redirect_uri is in the registered list
	for _, uri := range client.RedirectURIs {
		if uri == redirectURI {
			return nil
		}
	}

	return fmt.Errorf("redirect_uri not registered for this client")
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}
