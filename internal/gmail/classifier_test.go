package gmail

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	gmail "google.golang.org/api/gmail/v1"
)

// message builds a Gmail message carrying the given headers.
func message(headers ...*gmail.MessagePartHeader) *gmail.Message {
	return &gmail.Message{
		Payload: &gmail.MessagePart{Headers: headers},
	}
}

// header is a shorthand for a single Gmail header.
func header(name, value string) *gmail.MessagePartHeader {
	return &gmail.MessagePartHeader{Name: name, Value: value}
}

// threadWithReasons builds a thread with one message per X-GitHub-Reason value.
func threadWithReasons(reasons ...string) *gmail.Thread {
	t := &gmail.Thread{}
	for _, r := range reasons {
		t.Messages = append(t.Messages, message(header("X-GitHub-Reason", r)))
	}
	return t
}

func TestClassifyThread(t *testing.T) {
	tests := []struct {
		name       string
		thread     *gmail.Thread
		wantType   string
		wantRepo   string
		wantNumber string
	}{
		{
			name: "issue notification",
			thread: &gmail.Thread{Messages: []*gmail.Message{
				message(header("Message-ID", "<giantswarm/roadmap/issues/4321/1234567890@github.com>")),
			}},
			wantType:   "*gmail.GitHubIssue",
			wantRepo:   "giantswarm/roadmap",
			wantNumber: "4321",
		},
		{
			name: "pull request notification",
			thread: &gmail.Thread{Messages: []*gmail.Message{
				message(header("Message-ID", "<teemow/inboxfewer/pull/95/issue_event/1234567890@github.com>")),
			}},
			wantType:   "*gmail.GitHubPull",
			wantRepo:   "teemow/inboxfewer",
			wantNumber: "95",
		},
		{
			name: "classifies on a later message in the thread",
			thread: &gmail.Thread{Messages: []*gmail.Message{
				message(header("Subject", "no message id here")),
				message(header("Message-ID", "<teemow/inboxfewer/issues/7/9@github.com>")),
			}},
			wantType:   "*gmail.GitHubIssue",
			wantRepo:   "teemow/inboxfewer",
			wantNumber: "7",
		},
		{
			name: "non-GitHub thread",
			thread: &gmail.Thread{Messages: []*gmail.Message{
				message(header("Message-ID", "<CAB1234@mail.example.com>")),
			}},
			wantType: "<nil>",
		},
		{
			name:     "thread without messages",
			thread:   &gmail.Thread{},
			wantType: "<nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyThread(tt.thread, "user", "token")

			switch topic := got.(type) {
			case *GitHubIssue:
				assert.Equal(t, "*gmail.GitHubIssue", tt.wantType)
				assert.Equal(t, tt.wantRepo, topic.Repo)
				assert.Equal(t, tt.wantNumber, topic.Number)
				assert.Equal(t, "user", topic.GithubUser)
				assert.Equal(t, "token", topic.GithubToken)
			case *GitHubPull:
				assert.Equal(t, "*gmail.GitHubPull", tt.wantType)
				assert.Equal(t, tt.wantRepo, topic.Repo)
				assert.Equal(t, tt.wantNumber, topic.Number)
				assert.Equal(t, "user", topic.GithubUser)
				assert.Equal(t, "token", topic.GithubToken)
			default:
				assert.Equal(t, "<nil>", tt.wantType)
				assert.Nil(t, got)
			}
		})
	}
}

func TestThreadReasons(t *testing.T) {
	tests := []struct {
		name   string
		thread *gmail.Thread
		want   []string
	}{
		{
			name:   "single reason",
			thread: threadWithReasons("mention"),
			want:   []string{"mention"},
		},
		{
			name:   "keeps first-seen order and removes duplicates",
			thread: threadWithReasons("subscribed", "mention", "subscribed", "push"),
			want:   []string{"subscribed", "mention", "push"},
		},
		{
			name:   "lowercases and trims",
			thread: threadWithReasons("  Team_Mention  "),
			want:   []string{"team_mention"},
		},
		{
			name: "ignores messages without the header",
			thread: &gmail.Thread{Messages: []*gmail.Message{
				message(header("Subject", "hello")),
				message(header("X-GitHub-Reason", "author")),
				{Payload: nil},
			}},
			want: []string{"author"},
		},
		{
			name:   "thread without messages",
			thread: &gmail.Thread{},
			want:   nil,
		},
		{
			name:   "nil thread",
			thread: nil,
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ThreadReasons(tt.thread)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ThreadReasons() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPersonallyAddressed(t *testing.T) {
	tests := []struct {
		name    string
		thread  *gmail.Thread
		reasons []string
		want    bool
	}{
		{
			name:    "mentioned by handle",
			thread:  threadWithReasons("mention"),
			reasons: DefaultPersonalReasons,
			want:    true,
		},
		{
			name:    "assigned",
			thread:  threadWithReasons("assign"),
			reasons: DefaultPersonalReasons,
			want:    true,
		},
		{
			name:    "author and assignee",
			thread:  threadWithReasons("author", "assign"),
			reasons: DefaultPersonalReasons,
			want:    true,
		},
		{
			name:    "personal reason on only one message of the thread",
			thread:  threadWithReasons("subscribed", "push", "mention"),
			reasons: DefaultPersonalReasons,
			want:    true,
		},
		{
			name:    "team mention is not personal",
			thread:  threadWithReasons("team_mention"),
			reasons: DefaultPersonalReasons,
			want:    false,
		},
		{
			name:    "review request is not personal",
			thread:  threadWithReasons("review_requested", "push"),
			reasons: DefaultPersonalReasons,
			want:    false,
		},
		{
			name:    "header name casing is ignored",
			thread:  &gmail.Thread{Messages: []*gmail.Message{message(header("x-github-reason", "MENTION"))}},
			reasons: DefaultPersonalReasons,
			want:    true,
		},
		{
			name:    "thread without the header",
			thread:  &gmail.Thread{Messages: []*gmail.Message{message(header("Subject", "hello"))}},
			reasons: DefaultPersonalReasons,
			want:    false,
		},
		{
			name:    "no reasons configured protects nothing",
			thread:  threadWithReasons("mention"),
			reasons: nil,
			want:    false,
		},
		{
			name:    "nil thread",
			thread:  nil,
			reasons: DefaultPersonalReasons,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsPersonallyAddressed(tt.thread, tt.reasons))
		})
	}
}
