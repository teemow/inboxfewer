package drive

import (
	"testing"
	"time"

	drive "google.golang.org/api/drive/v3"
)

func TestConvertToFileInfo(t *testing.T) {
	createdTime := "2023-01-01T10:00:00Z"
	modifiedTime := "2023-01-02T15:30:00Z"
	trashedTime := "2023-01-03T20:00:00Z"

	driveFile := &drive.File{
		Id:             "file123",
		Name:           "test.pdf",
		MimeType:       "application/pdf",
		Size:           1024,
		CreatedTime:    createdTime,
		ModifiedTime:   modifiedTime,
		TrashedTime:    trashedTime,
		WebViewLink:    "https://drive.google.com/file/d/file123/view",
		WebContentLink: "https://drive.google.com/uc?id=file123",
		Parents:        []string{"parent1", "parent2"},
		Shared:         true,
		Trashed:        true,
		Owners: []*drive.User{
			{
				DisplayName:  "Test User",
				EmailAddress: "test@example.com",
				PhotoLink:    "https://example.com/photo.jpg",
			},
		},
		Permissions: []*drive.Permission{
			{
				Id:           "perm123",
				Type:         "user",
				Role:         "reader",
				EmailAddress: "reader@example.com",
				DisplayName:  "Reader User",
			},
		},
	}

	fileInfo := convertToFileInfo(driveFile)

	// Test basic fields
	if fileInfo.ID != "file123" {
		t.Errorf("Expected ID file123, got %s", fileInfo.ID)
	}
	if fileInfo.Name != "test.pdf" {
		t.Errorf("Expected Name test.pdf, got %s", fileInfo.Name)
	}
	if fileInfo.MimeType != "application/pdf" {
		t.Errorf("Expected MimeType application/pdf, got %s", fileInfo.MimeType)
	}
	if fileInfo.Size != 1024 {
		t.Errorf("Expected Size 1024, got %d", fileInfo.Size)
	}
	if fileInfo.WebViewLink != "https://drive.google.com/file/d/file123/view" {
		t.Errorf("Expected WebViewLink, got %s", fileInfo.WebViewLink)
	}
	if fileInfo.WebContentLink != "https://drive.google.com/uc?id=file123" {
		t.Errorf("Expected WebContentLink, got %s", fileInfo.WebContentLink)
	}
	if !fileInfo.Shared {
		t.Error("Expected Shared to be true")
	}
	if !fileInfo.Trashed {
		t.Error("Expected Trashed to be true")
	}

	// Test parents
	if len(fileInfo.Parents) != 2 {
		t.Errorf("Expected 2 parents, got %d", len(fileInfo.Parents))
	}
	if fileInfo.Parents[0] != "parent1" || fileInfo.Parents[1] != "parent2" {
		t.Errorf("Expected parents [parent1, parent2], got %v", fileInfo.Parents)
	}

	// Test timestamps
	expectedCreated, _ := time.Parse(time.RFC3339, createdTime)
	if !fileInfo.CreatedTime.Equal(expectedCreated) {
		t.Errorf("Expected CreatedTime %v, got %v", expectedCreated, fileInfo.CreatedTime)
	}

	expectedModified, _ := time.Parse(time.RFC3339, modifiedTime)
	if !fileInfo.ModifiedTime.Equal(expectedModified) {
		t.Errorf("Expected ModifiedTime %v, got %v", expectedModified, fileInfo.ModifiedTime)
	}

	if fileInfo.TrashedTime == nil {
		t.Error("Expected TrashedTime to be set")
	} else {
		expectedTrashed, _ := time.Parse(time.RFC3339, trashedTime)
		if !fileInfo.TrashedTime.Equal(expectedTrashed) {
			t.Errorf("Expected TrashedTime %v, got %v", expectedTrashed, *fileInfo.TrashedTime)
		}
	}

	// Test owners
	if len(fileInfo.Owners) != 1 {
		t.Errorf("Expected 1 owner, got %d", len(fileInfo.Owners))
	} else {
		owner := fileInfo.Owners[0]
		if owner.DisplayName != "Test User" {
			t.Errorf("Expected owner DisplayName 'Test User', got %s", owner.DisplayName)
		}
		if owner.EmailAddress != "test@example.com" {
			t.Errorf("Expected owner EmailAddress 'test@example.com', got %s", owner.EmailAddress)
		}
		if owner.PhotoLink != "https://example.com/photo.jpg" {
			t.Errorf("Expected owner PhotoLink, got %s", owner.PhotoLink)
		}
	}

	// Test permissions
	if len(fileInfo.Permissions) != 1 {
		t.Errorf("Expected 1 permission, got %d", len(fileInfo.Permissions))
	} else {
		perm := fileInfo.Permissions[0]
		if perm.ID != "perm123" {
			t.Errorf("Expected permission ID perm123, got %s", perm.ID)
		}
		if perm.Type != "user" {
			t.Errorf("Expected permission Type user, got %s", perm.Type)
		}
		if perm.Role != "reader" {
			t.Errorf("Expected permission Role reader, got %s", perm.Role)
		}
		if perm.EmailAddress != "reader@example.com" {
			t.Errorf("Expected permission EmailAddress reader@example.com, got %s", perm.EmailAddress)
		}
		if perm.DisplayName != "Reader User" {
			t.Errorf("Expected permission DisplayName 'Reader User', got %s", perm.DisplayName)
		}
	}
}

func TestConvertToFileInfo_MinimalData(t *testing.T) {
	driveFile := &drive.File{
		Id:       "file456",
		Name:     "minimal.txt",
		MimeType: "text/plain",
	}

	fileInfo := convertToFileInfo(driveFile)

	if fileInfo.ID != "file456" {
		t.Errorf("Expected ID file456, got %s", fileInfo.ID)
	}
	if fileInfo.Name != "minimal.txt" {
		t.Errorf("Expected Name minimal.txt, got %s", fileInfo.Name)
	}
	if fileInfo.MimeType != "text/plain" {
		t.Errorf("Expected MimeType text/plain, got %s", fileInfo.MimeType)
	}
	if fileInfo.Size != 0 {
		t.Errorf("Expected Size 0, got %d", fileInfo.Size)
	}
	if len(fileInfo.Owners) != 0 {
		t.Errorf("Expected 0 owners, got %d", len(fileInfo.Owners))
	}
	if len(fileInfo.Permissions) != 0 {
		t.Errorf("Expected 0 permissions, got %d", len(fileInfo.Permissions))
	}
}

func TestConvertToPermission(t *testing.T) {
	drivePermission := &drive.Permission{
		Id:           "perm456",
		Type:         "group",
		Role:         "writer",
		EmailAddress: "group@example.com",
		Domain:       "example.com",
		DisplayName:  "Example Group",
	}

	permission := convertToPermission(drivePermission)

	if permission.ID != "perm456" {
		t.Errorf("Expected ID perm456, got %s", permission.ID)
	}
	if permission.Type != "group" {
		t.Errorf("Expected Type group, got %s", permission.Type)
	}
	if permission.Role != "writer" {
		t.Errorf("Expected Role writer, got %s", permission.Role)
	}
	if permission.EmailAddress != "group@example.com" {
		t.Errorf("Expected EmailAddress group@example.com, got %s", permission.EmailAddress)
	}
	if permission.Domain != "example.com" {
		t.Errorf("Expected Domain example.com, got %s", permission.Domain)
	}
	if permission.DisplayName != "Example Group" {
		t.Errorf("Expected DisplayName 'Example Group', got %s", permission.DisplayName)
	}
}

func TestConvertToPermission_MinimalData(t *testing.T) {
	drivePermission := &drive.Permission{
		Id:   "perm789",
		Type: "anyone",
		Role: "reader",
	}

	permission := convertToPermission(drivePermission)

	if permission.ID != "perm789" {
		t.Errorf("Expected ID perm789, got %s", permission.ID)
	}
	if permission.Type != "anyone" {
		t.Errorf("Expected Type anyone, got %s", permission.Type)
	}
	if permission.Role != "reader" {
		t.Errorf("Expected Role reader, got %s", permission.Role)
	}
	if permission.EmailAddress != "" {
		t.Errorf("Expected empty EmailAddress, got %s", permission.EmailAddress)
	}
	if permission.Domain != "" {
		t.Errorf("Expected empty Domain, got %s", permission.Domain)
	}
	if permission.DisplayName != "" {
		t.Errorf("Expected empty DisplayName, got %s", permission.DisplayName)
	}
}

func TestAccount(t *testing.T) {
	client := &Client{
		account: "test-account",
	}

	if client.Account() != "test-account" {
		t.Errorf("Expected account 'test-account', got %s", client.Account())
	}
}

func TestHasToken(t *testing.T) {
	// This test just ensures the functions exist and can be called
	// Actual functionality is tested in the google package
	_ = HasToken()
	_ = HasTokenForAccount("test")
}

func TestFolderMimeType(t *testing.T) {
	expectedMimeType := "application/vnd.google-apps.folder"
	if FolderMimeType != expectedMimeType {
		t.Errorf("Expected FolderMimeType %s, got %s", expectedMimeType, FolderMimeType)
	}
}

// TestBuildListFilesQuery tests the query building logic for listing files
func TestBuildListFilesQuery(t *testing.T) {
	tests := []struct {
		name           string
		userQuery      string
		includeTrashed bool
		expected       string
	}{
		{
			name:           "user query with trashed excluded (default)",
			userQuery:      "mimeType='application/pdf'",
			includeTrashed: false,
			expected:       "(mimeType='application/pdf') and trashed=false",
		},
		{
			name:           "user query with trashed included",
			userQuery:      "mimeType='application/pdf'",
			includeTrashed: true,
			expected:       "mimeType='application/pdf'",
		},
		{
			name:           "no user query, exclude trashed (default)",
			userQuery:      "",
			includeTrashed: false,
			expected:       "trashed=false",
		},
		{
			name:           "no user query, include trashed",
			userQuery:      "",
			includeTrashed: true,
			expected:       "",
		},
		{
			name:           "complex query with name filter",
			userQuery:      "name contains 'house' or name contains 'water'",
			includeTrashed: false,
			expected:       "(name contains 'house' or name contains 'water') and trashed=false",
		},
		{
			name:           "query for folders only",
			userQuery:      "mimeType='application/vnd.google-apps.folder'",
			includeTrashed: false,
			expected:       "(mimeType='application/vnd.google-apps.folder') and trashed=false",
		},
		{
			name:           "query with multiple conditions",
			userQuery:      "mimeType='application/pdf' and name contains 'report'",
			includeTrashed: false,
			expected:       "(mimeType='application/pdf' and name contains 'report') and trashed=false",
		},
		{
			name:           "query with parentheses",
			userQuery:      "(mimeType='application/pdf' or mimeType='image/jpeg') and starred=true",
			includeTrashed: false,
			expected:       "((mimeType='application/pdf' or mimeType='image/jpeg') and starred=true) and trashed=false",
		},
		{
			name:           "query for owned files",
			userQuery:      "'me' in owners",
			includeTrashed: false,
			expected:       "('me' in owners) and trashed=false",
		},
		{
			name:           "query with date filter",
			userQuery:      "modifiedTime > '2025-01-01T00:00:00'",
			includeTrashed: false,
			expected:       "(modifiedTime > '2025-01-01T00:00:00') and trashed=false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildListFilesQuery(tt.userQuery, tt.includeTrashed)
			if result != tt.expected {
				t.Errorf("buildListFilesQuery(%q, %v) = %q, want %q",
					tt.userQuery, tt.includeTrashed, result, tt.expected)
			}
		})
	}
}
