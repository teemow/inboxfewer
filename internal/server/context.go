package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/teemow/inboxfewer/internal/docs"
	"github.com/teemow/inboxfewer/internal/gmail"
)

// ServerContext holds the context for the MCP server
type ServerContext struct {
	ctx          context.Context
	cancel       context.CancelFunc
	gmailClients map[string]*gmail.Client // Maps account name to Gmail client
	docsClients  map[string]*docs.Client  // Maps account name to Docs client
	githubUser   string
	githubToken  string
	mu           sync.RWMutex
	shutdown     bool
}

// NewServerContext creates a new server context
func NewServerContext(ctx context.Context, githubUser, githubToken string) (*ServerContext, error) {
	shutdownCtx, cancel := context.WithCancel(ctx)

	// Initialize client maps
	gmailClients := make(map[string]*gmail.Client)
	docsClients := make(map[string]*docs.Client)

	// Try to create default Gmail client, but don't fail if token is missing
	// Clients will be lazily initialized when first needed
	if gmail.HasToken() {
		gmailClient, err := gmail.NewClient(shutdownCtx)
		if err != nil {
			// Log but don't fail - will be re-attempted on first use
			fmt.Printf("Warning: failed to create Gmail client for default account: %v\n", err)
		} else {
			gmailClients["default"] = gmailClient
		}
	}

	return &ServerContext{
		ctx:          shutdownCtx,
		cancel:       cancel,
		gmailClients: gmailClients,
		docsClients:  docsClients,
		githubUser:   githubUser,
		githubToken:  githubToken,
		shutdown:     false,
	}, nil
}

// Context returns the server context
func (sc *ServerContext) Context() context.Context {
	return sc.ctx
}

// GmailClientForAccount returns the Gmail client for a specific account
// Creates and caches the client if it doesn't exist yet
// Returns nil if the account has no token
func (sc *ServerContext) GmailClientForAccount(account string) *gmail.Client {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Check if client already exists
	if client, ok := sc.gmailClients[account]; ok {
		return client
	}

	// Try to create client if token exists
	if !gmail.HasTokenForAccount(account) {
		return nil
	}

	client, err := gmail.NewClientForAccount(sc.ctx, account)
	if err != nil {
		fmt.Printf("Warning: failed to create Gmail client for account %s: %v\n", account, err)
		return nil
	}

	sc.gmailClients[account] = client
	return client
}

// GmailClient returns the Gmail client for the default account
func (sc *ServerContext) GmailClient() *gmail.Client {
	return sc.GmailClientForAccount("default")
}

// SetGmailClientForAccount sets the Gmail client for a specific account
func (sc *ServerContext) SetGmailClientForAccount(account string, client *gmail.Client) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.gmailClients[account] = client
}

// SetGmailClient sets the Gmail client for the default account
func (sc *ServerContext) SetGmailClient(client *gmail.Client) {
	sc.SetGmailClientForAccount("default", client)
}

// DocsClientForAccount returns the Docs client for a specific account
// Creates and caches the client if it doesn't exist yet
// Returns nil if the account has no token
func (sc *ServerContext) DocsClientForAccount(account string) *docs.Client {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Check if client already exists
	if client, ok := sc.docsClients[account]; ok {
		return client
	}

	// Try to create client if token exists
	if !docs.HasTokenForAccount(account) {
		return nil
	}

	client, err := docs.NewClientForAccount(sc.ctx, account)
	if err != nil {
		fmt.Printf("Warning: failed to create Docs client for account %s: %v\n", account, err)
		return nil
	}

	sc.docsClients[account] = client
	return client
}

// DocsClient returns the Docs client for the default account
func (sc *ServerContext) DocsClient() *docs.Client {
	return sc.DocsClientForAccount("default")
}

// SetDocsClientForAccount sets the Docs client for a specific account
func (sc *ServerContext) SetDocsClientForAccount(account string, client *docs.Client) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.docsClients[account] = client
}

// SetDocsClient sets the Docs client for the default account
func (sc *ServerContext) SetDocsClient(client *docs.Client) {
	sc.SetDocsClientForAccount("default", client)
}

// GithubUser returns the GitHub username
func (sc *ServerContext) GithubUser() string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.githubUser
}

// GithubToken returns the GitHub token
func (sc *ServerContext) GithubToken() string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.githubToken
}

// IsShutdown returns whether the server has been shutdown
func (sc *ServerContext) IsShutdown() bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.shutdown
}

// Shutdown shuts down the server context
func (sc *ServerContext) Shutdown() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.shutdown {
		return nil
	}

	sc.shutdown = true
	sc.cancel()
	return nil
}
