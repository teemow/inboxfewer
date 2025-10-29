package docs

// DocumentMetadata represents metadata about a Google Drive file
type DocumentMetadata struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MimeType     string `json:"mimeType"`
	CreatedTime  string `json:"createdTime"`
	ModifiedTime string `json:"modifiedTime"`
	Size         int64  `json:"size,omitempty"`
	Owners       []User `json:"owners,omitempty"`
}

// User represents a Google Drive user
type User struct {
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
}
