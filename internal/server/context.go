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
	ctx         context.Context
	cancel      context.CancelFunc
	gmailClient *gmail.Client
	docsClient  *docs.Client
	githubUser  string
	githubToken string
	mu          sync.RWMutex
	shutdown    bool
}

// NewServerContext creates a new server context
func NewServerContext(ctx context.Context, githubUser, githubToken string) (*ServerContext, error) {
	shutdownCtx, cancel := context.WithCancel(ctx)

	// Create Gmail client
	gmailClient, err := gmail.NewClient(shutdownCtx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create Gmail client: %w", err)
	}

	return &ServerContext{
		ctx:         shutdownCtx,
		cancel:      cancel,
		gmailClient: gmailClient,
		githubUser:  githubUser,
		githubToken: githubToken,
		shutdown:    false,
	}, nil
}

// Context returns the server context
func (sc *ServerContext) Context() context.Context {
	return sc.ctx
}

// GmailClient returns the Gmail client
func (sc *ServerContext) GmailClient() *gmail.Client {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.gmailClient
}

// DocsClient returns the Docs client (lazy initialization)
func (sc *ServerContext) DocsClient() *docs.Client {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.docsClient
}

// SetDocsClient sets the Docs client
func (sc *ServerContext) SetDocsClient(client *docs.Client) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.docsClient = client
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
