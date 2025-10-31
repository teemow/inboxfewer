package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/teemow/inboxfewer/internal/calendar"
	"github.com/teemow/inboxfewer/internal/docs"
	"github.com/teemow/inboxfewer/internal/gmail"
	"github.com/teemow/inboxfewer/internal/meet"
	"github.com/teemow/inboxfewer/internal/tasks"
)

// ServerContext holds the context for the MCP server
type ServerContext struct {
	ctx             context.Context
	cancel          context.CancelFunc
	gmailClients    map[string]*gmail.Client    // Maps account name to Gmail client
	docsClients     map[string]*docs.Client     // Maps account name to Docs client
	calendarClients map[string]*calendar.Client // Maps account name to Calendar client
	meetClients     map[string]*meet.Client     // Maps account name to Meet client
	tasksClients    map[string]*tasks.Client    // Maps account name to Tasks client
	githubUser      string
	githubToken     string
	mu              sync.RWMutex
	shutdown        bool
}

// NewServerContext creates a new server context
func NewServerContext(ctx context.Context, githubUser, githubToken string) (*ServerContext, error) {
	shutdownCtx, cancel := context.WithCancel(ctx)

	// Initialize client maps
	gmailClients := make(map[string]*gmail.Client)
	docsClients := make(map[string]*docs.Client)
	calendarClients := make(map[string]*calendar.Client)
	meetClients := make(map[string]*meet.Client)
	tasksClients := make(map[string]*tasks.Client)

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
		ctx:             shutdownCtx,
		cancel:          cancel,
		gmailClients:    gmailClients,
		docsClients:     docsClients,
		calendarClients: calendarClients,
		meetClients:     meetClients,
		tasksClients:    tasksClients,
		githubUser:      githubUser,
		githubToken:     githubToken,
		shutdown:        false,
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

// CalendarClientForAccount returns the Calendar client for a specific account
// Creates and caches the client if it doesn't exist yet
// Returns nil if the account has no token
func (sc *ServerContext) CalendarClientForAccount(account string) *calendar.Client {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Check if client already exists
	if client, ok := sc.calendarClients[account]; ok {
		return client
	}

	// Try to create client if token exists
	if !calendar.HasTokenForAccount(account) {
		return nil
	}

	client, err := calendar.NewClientForAccount(sc.ctx, account)
	if err != nil {
		fmt.Printf("Warning: failed to create Calendar client for account %s: %v\n", account, err)
		return nil
	}

	sc.calendarClients[account] = client
	return client
}

// CalendarClient returns the Calendar client for the default account
func (sc *ServerContext) CalendarClient() *calendar.Client {
	return sc.CalendarClientForAccount("default")
}

// SetCalendarClientForAccount sets the Calendar client for a specific account
func (sc *ServerContext) SetCalendarClientForAccount(account string, client *calendar.Client) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.calendarClients[account] = client
}

// SetCalendarClient sets the Calendar client for the default account
func (sc *ServerContext) SetCalendarClient(client *calendar.Client) {
	sc.SetCalendarClientForAccount("default", client)
}

// MeetClientForAccount returns the Meet client for a specific account
// Creates and caches the client if it doesn't exist yet
// Returns nil if the account has no token
func (sc *ServerContext) MeetClientForAccount(account string) *meet.Client {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Check if client already exists
	if client, ok := sc.meetClients[account]; ok {
		return client
	}

	// Try to create client if token exists
	if !meet.HasTokenForAccount(account) {
		return nil
	}

	client, err := meet.NewClientForAccount(sc.ctx, account)
	if err != nil {
		fmt.Printf("Warning: failed to create Meet client for account %s: %v\n", account, err)
		return nil
	}

	sc.meetClients[account] = client
	return client
}

// MeetClient returns the Meet client for the default account
func (sc *ServerContext) MeetClient() *meet.Client {
	return sc.MeetClientForAccount("default")
}

// SetMeetClientForAccount sets the Meet client for a specific account
func (sc *ServerContext) SetMeetClientForAccount(account string, client *meet.Client) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.meetClients[account] = client
}

// SetMeetClient sets the Meet client for the default account
func (sc *ServerContext) SetMeetClient(client *meet.Client) {
	sc.SetMeetClientForAccount("default", client)
}

// TasksClientForAccount returns the Tasks client for a specific account
// Creates and caches the client if it doesn't exist yet
// Returns nil if the account has no token
func (sc *ServerContext) TasksClientForAccount(account string) *tasks.Client {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Check if client already exists
	if client, ok := sc.tasksClients[account]; ok {
		return client
	}

	// Try to create client if token exists
	if !tasks.HasTokenForAccount(account) {
		return nil
	}

	client, err := tasks.NewClientForAccount(sc.ctx, account)
	if err != nil {
		fmt.Printf("Warning: failed to create Tasks client for account %s: %v\n", account, err)
		return nil
	}

	sc.tasksClients[account] = client
	return client
}

// TasksClient returns the Tasks client for the default account
func (sc *ServerContext) TasksClient() *tasks.Client {
	return sc.TasksClientForAccount("default")
}

// SetTasksClientForAccount sets the Tasks client for a specific account
func (sc *ServerContext) SetTasksClientForAccount(account string, client *tasks.Client) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.tasksClients[account] = client
}

// SetTasksClient sets the Tasks client for the default account
func (sc *ServerContext) SetTasksClient(client *tasks.Client) {
	sc.SetTasksClientForAccount("default", client)
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
