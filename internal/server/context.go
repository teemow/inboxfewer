package server

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/teemow/inboxfewer/internal/calendar"
	"github.com/teemow/inboxfewer/internal/docs"
	"github.com/teemow/inboxfewer/internal/drive"
	"github.com/teemow/inboxfewer/internal/gmail"
	"github.com/teemow/inboxfewer/internal/google"
	"github.com/teemow/inboxfewer/internal/instrumentation"
	"github.com/teemow/inboxfewer/internal/meet"
	"github.com/teemow/inboxfewer/internal/tasks"
)

// clientFactory is a function that creates a new Google API client for a given account
type clientFactory[T any] func(ctx context.Context, account string, provider google.TokenProvider) (T, error)

// clientCache manages lazy-initialized, cached Google API clients per account
type clientCache[T any] struct {
	clients map[string]T
	factory clientFactory[T]
	name    string // Service name for logging (e.g., "Gmail", "Docs")
}

// newClientCache creates a new client cache with the given factory
func newClientCache[T any](factory clientFactory[T], name string) *clientCache[T] {
	return &clientCache[T]{
		clients: make(map[string]T),
		factory: factory,
		name:    name,
	}
}

// get retrieves or creates a client for the given account
// Returns the zero value of T if the account has no token or client creation fails
func (c *clientCache[T]) get(ctx context.Context, account string, provider google.TokenProvider, logger *slog.Logger, mu *sync.RWMutex) T {
	mu.Lock()
	defer mu.Unlock()

	// Check if client already exists
	if client, ok := c.clients[account]; ok {
		return client
	}

	// Try to create client if token exists
	if provider == nil || !provider.HasTokenForAccount(account) {
		var zero T
		return zero
	}

	client, err := c.factory(ctx, account, provider)
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to create %s client", c.name), "account", account, "error", err)
		var zero T
		return zero
	}

	c.clients[account] = client
	return client
}

// set stores a client for the given account
func (c *clientCache[T]) set(account string, client T, mu *sync.RWMutex) {
	mu.Lock()
	defer mu.Unlock()
	c.clients[account] = client
}

// ServerContext holds the context for the MCP server
type ServerContext struct {
	ctx           context.Context
	cancel        context.CancelFunc
	gmailCache    *clientCache[*gmail.Client]
	docsCache     *clientCache[*docs.Client]
	driveCache    *clientCache[*drive.Client]
	calendarCache *clientCache[*calendar.Client]
	meetCache     *clientCache[*meet.Client]
	tasksCache    *clientCache[*tasks.Client]
	githubUser    string
	githubToken   string
	tokenProvider google.TokenProvider // Token provider for Google API authentication
	logger        *slog.Logger
	metrics       *instrumentation.Metrics     // Metrics recorder for observability
	auditLogger   *instrumentation.AuditLogger // Audit logger for tool invocations
	mu            sync.RWMutex
	shutdown      bool
}

// NewServerContext creates a new server context with file-based token provider (for STDIO transport)
func NewServerContext(ctx context.Context, githubUser, githubToken string) (*ServerContext, error) {
	return NewServerContextWithProvider(ctx, githubUser, githubToken, google.NewFileTokenProvider())
}

// NewServerContextWithProvider creates a new server context with a custom token provider (for HTTP/SSE transport)
func NewServerContextWithProvider(ctx context.Context, githubUser, githubToken string, tokenProvider google.TokenProvider) (*ServerContext, error) {
	shutdownCtx, cancel := context.WithCancel(ctx)

	logger := slog.Default()

	// Initialize client caches with their respective factories
	sc := &ServerContext{
		ctx:           shutdownCtx,
		cancel:        cancel,
		gmailCache:    newClientCache(gmail.NewClientForAccountWithProvider, "Gmail"),
		docsCache:     newClientCache(docs.NewClientForAccountWithProvider, "Docs"),
		driveCache:    newClientCache(drive.NewClientForAccountWithProvider, "Drive"),
		calendarCache: newClientCache(calendar.NewClientForAccountWithProvider, "Calendar"),
		meetCache:     newClientCache(meet.NewClientForAccountWithProvider, "Meet"),
		tasksCache:    newClientCache(tasks.NewClientForAccountWithProvider, "Tasks"),
		githubUser:    githubUser,
		githubToken:   githubToken,
		tokenProvider: tokenProvider,
		logger:        logger,
		shutdown:      false,
	}

	// Try to eagerly create default Gmail client if token provider has a token
	// but don't fail if token is missing - clients will be lazily initialized when needed
	if tokenProvider != nil && tokenProvider.HasTokenForAccount("default") {
		gmailClient, err := gmail.NewClientWithProvider(shutdownCtx, tokenProvider)
		if err != nil {
			// Log but don't fail - will be re-attempted on first use
			logger.Warn("Failed to create Gmail client for default account", "error", err)
		} else {
			sc.gmailCache.set("default", gmailClient, &sc.mu)
		}
	}

	return sc, nil
}

// Context returns the server context
func (sc *ServerContext) Context() context.Context {
	return sc.ctx
}

// GmailClientForAccount returns the Gmail client for a specific account
// Creates and caches the client if it doesn't exist yet
// Returns nil if the account has no token
func (sc *ServerContext) GmailClientForAccount(account string) *gmail.Client {
	return sc.gmailCache.get(sc.ctx, account, sc.tokenProvider, sc.logger, &sc.mu)
}

// GmailClient returns the Gmail client for the default account
func (sc *ServerContext) GmailClient() *gmail.Client {
	return sc.GmailClientForAccount("default")
}

// SetGmailClientForAccount sets the Gmail client for a specific account
func (sc *ServerContext) SetGmailClientForAccount(account string, client *gmail.Client) {
	sc.gmailCache.set(account, client, &sc.mu)
}

// SetGmailClient sets the Gmail client for the default account
func (sc *ServerContext) SetGmailClient(client *gmail.Client) {
	sc.SetGmailClientForAccount("default", client)
}

// DocsClientForAccount returns the Docs client for a specific account
// Creates and caches the client if it doesn't exist yet
// Returns nil if the account has no token
func (sc *ServerContext) DocsClientForAccount(account string) *docs.Client {
	return sc.docsCache.get(sc.ctx, account, sc.tokenProvider, sc.logger, &sc.mu)
}

// DocsClient returns the Docs client for the default account
func (sc *ServerContext) DocsClient() *docs.Client {
	return sc.DocsClientForAccount("default")
}

// SetDocsClientForAccount sets the Docs client for a specific account
func (sc *ServerContext) SetDocsClientForAccount(account string, client *docs.Client) {
	sc.docsCache.set(account, client, &sc.mu)
}

// SetDocsClient sets the Docs client for the default account
func (sc *ServerContext) SetDocsClient(client *docs.Client) {
	sc.SetDocsClientForAccount("default", client)
}

// CalendarClientForAccount returns the Calendar client for a specific account
// Creates and caches the client if it doesn't exist yet
// Returns nil if the account has no token
func (sc *ServerContext) CalendarClientForAccount(account string) *calendar.Client {
	return sc.calendarCache.get(sc.ctx, account, sc.tokenProvider, sc.logger, &sc.mu)
}

// CalendarClient returns the Calendar client for the default account
func (sc *ServerContext) CalendarClient() *calendar.Client {
	return sc.CalendarClientForAccount("default")
}

// SetCalendarClientForAccount sets the Calendar client for a specific account
func (sc *ServerContext) SetCalendarClientForAccount(account string, client *calendar.Client) {
	sc.calendarCache.set(account, client, &sc.mu)
}

// SetCalendarClient sets the Calendar client for the default account
func (sc *ServerContext) SetCalendarClient(client *calendar.Client) {
	sc.SetCalendarClientForAccount("default", client)
}

// MeetClientForAccount returns the Meet client for a specific account
// Creates and caches the client if it doesn't exist yet
// Returns nil if the account has no token
func (sc *ServerContext) MeetClientForAccount(account string) *meet.Client {
	return sc.meetCache.get(sc.ctx, account, sc.tokenProvider, sc.logger, &sc.mu)
}

// MeetClient returns the Meet client for the default account
func (sc *ServerContext) MeetClient() *meet.Client {
	return sc.MeetClientForAccount("default")
}

// SetMeetClientForAccount sets the Meet client for a specific account
func (sc *ServerContext) SetMeetClientForAccount(account string, client *meet.Client) {
	sc.meetCache.set(account, client, &sc.mu)
}

// SetMeetClient sets the Meet client for the default account
func (sc *ServerContext) SetMeetClient(client *meet.Client) {
	sc.SetMeetClientForAccount("default", client)
}

// TasksClientForAccount returns the Tasks client for a specific account
// Creates and caches the client if it doesn't exist yet
// Returns nil if the account has no token
func (sc *ServerContext) TasksClientForAccount(account string) *tasks.Client {
	return sc.tasksCache.get(sc.ctx, account, sc.tokenProvider, sc.logger, &sc.mu)
}

// TasksClient returns the Tasks client for the default account
func (sc *ServerContext) TasksClient() *tasks.Client {
	return sc.TasksClientForAccount("default")
}

// SetTasksClientForAccount sets the Tasks client for a specific account
func (sc *ServerContext) SetTasksClientForAccount(account string, client *tasks.Client) {
	sc.tasksCache.set(account, client, &sc.mu)
}

// SetTasksClient sets the Tasks client for the default account
func (sc *ServerContext) SetTasksClient(client *tasks.Client) {
	sc.SetTasksClientForAccount("default", client)
}

// DriveClientForAccount returns the Drive client for a specific account
// Creates and caches the client if it doesn't exist yet
// Returns nil if the account has no token
func (sc *ServerContext) DriveClientForAccount(account string) *drive.Client {
	return sc.driveCache.get(sc.ctx, account, sc.tokenProvider, sc.logger, &sc.mu)
}

// DriveClient returns the Drive client for the default account
func (sc *ServerContext) DriveClient() *drive.Client {
	return sc.DriveClientForAccount("default")
}

// SetDriveClientForAccount sets the Drive client for a specific account
func (sc *ServerContext) SetDriveClientForAccount(account string, client *drive.Client) {
	sc.driveCache.set(account, client, &sc.mu)
}

// SetDriveClient sets the Drive client for the default account
func (sc *ServerContext) SetDriveClient(client *drive.Client) {
	sc.SetDriveClientForAccount("default", client)
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

// Metrics returns the metrics recorder for observability.
// Returns nil if metrics are not configured.
func (sc *ServerContext) Metrics() *instrumentation.Metrics {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.metrics
}

// SetMetrics sets the metrics recorder for the server context.
func (sc *ServerContext) SetMetrics(metrics *instrumentation.Metrics) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.metrics = metrics
}

// AuditLogger returns the audit logger for tool invocations.
// Returns nil if audit logging is not configured.
func (sc *ServerContext) AuditLogger() *instrumentation.AuditLogger {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.auditLogger
}

// SetAuditLogger sets the audit logger for the server context.
func (sc *ServerContext) SetAuditLogger(logger *instrumentation.AuditLogger) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.auditLogger = logger
}

// Logger returns the slog logger.
func (sc *ServerContext) Logger() *slog.Logger {
	return sc.logger
}
