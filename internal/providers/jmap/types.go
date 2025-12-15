// Package jmap provides JMAP (JSON Meta Application Protocol) implementation
// for the mailboxzero email client.
package jmap

import "time"

// Session represents a JMAP session with server information
type Session struct {
	Username        string                 `json:"username"`
	APIUrl          string                 `json:"apiUrl"`
	DownloadUrl     string                 `json:"downloadUrl"`
	UploadUrl       string                 `json:"uploadUrl"`
	EventSourceUrl  string                 `json:"eventSourceUrl"`
	State           string                 `json:"state"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	Accounts        map[string]Account     `json:"accounts"`
	PrimaryAccounts map[string]string      `json:"primaryAccounts"`
}

// Account represents a JMAP account
type Account struct {
	Name                string                 `json:"name"`
	IsPersonal          bool                   `json:"isPersonal"`
	IsReadOnly          bool                   `json:"isReadOnly"`
	AccountCapabilities map[string]interface{} `json:"accountCapabilities"`
}

// MethodCall represents a JMAP method call (array of interface{})
type MethodCall []interface{}

// Response represents a JMAP response
type Response struct {
	MethodResponses [][]interface{} `json:"methodResponses"`
	SessionState    string          `json:"sessionState"`
}

// jmapEmail represents the internal JMAP email structure
// This is kept separate from protocol.Email to handle JMAP-specific fields
type jmapEmail struct {
	ID            string               `json:"id"`
	BlobID        string               `json:"blobId"`
	ThreadID      string               `json:"threadId"`
	MailboxIDs    map[string]bool      `json:"mailboxIds"`
	Keywords      map[string]bool      `json:"keywords"`
	Size          int                  `json:"size"`
	ReceivedAt    time.Time            `json:"receivedAt"`
	MessageID     []string             `json:"messageId"`
	InReplyTo     []string             `json:"inReplyTo"`
	References    []string             `json:"references"`
	Sender        []jmapEmailAddress   `json:"sender"`
	From          []jmapEmailAddress   `json:"from"`
	To            []jmapEmailAddress   `json:"to"`
	Cc            []jmapEmailAddress   `json:"cc"`
	Bcc           []jmapEmailAddress   `json:"bcc"`
	ReplyTo       []jmapEmailAddress   `json:"replyTo"`
	Subject       string               `json:"subject"`
	SentAt        time.Time            `json:"sentAt"`
	HasAttachment bool                 `json:"hasAttachment"`
	Preview       string               `json:"preview"`
	BodyValues    map[string]BodyValue `json:"bodyValues"`
	TextBody      []BodyPart           `json:"textBody"`
	HTMLBody      []BodyPart           `json:"htmlBody"`
	Attachments   []Attachment         `json:"attachments"`
}

// jmapEmailAddress represents a JMAP email address
type jmapEmailAddress struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// BodyValue represents JMAP body content
type BodyValue struct {
	Value             string `json:"value"`
	IsEncodingProblem bool   `json:"isEncodingProblem"`
	IsTruncated       bool   `json:"isTruncated"`
}

// BodyPart represents a JMAP body part
type BodyPart struct {
	PartID      string            `json:"partId"`
	BlobID      string            `json:"blobId"`
	Size        int               `json:"size"`
	Headers     map[string]string `json:"headers"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Charset     string            `json:"charset"`
	Disposition string            `json:"disposition"`
	CID         string            `json:"cid"`
	Language    []string          `json:"language"`
	Location    string            `json:"location"`
	SubParts    []BodyPart        `json:"subParts"`
}

// Attachment represents a JMAP attachment
type Attachment struct {
	PartID      string            `json:"partId"`
	BlobID      string            `json:"blobId"`
	Size        int               `json:"size"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Charset     string            `json:"charset"`
	Disposition string            `json:"disposition"`
	CID         string            `json:"cid"`
	Headers     map[string]string `json:"headers"`
}

// Mailbox represents a JMAP mailbox
type Mailbox struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ParentID      string `json:"parentId"`
	Role          string `json:"role"`
	SortOrder     int    `json:"sortOrder"`
	TotalEmails   int    `json:"totalEmails"`
	UnreadEmails  int    `json:"unreadEmails"`
	TotalThreads  int    `json:"totalThreads"`
	UnreadThreads int    `json:"unreadThreads"`
	MyRights      Rights `json:"myRights"`
	IsSubscribed  bool   `json:"isSubscribed"`
}

// Rights represents JMAP mailbox rights
type Rights struct {
	MayReadItems   bool `json:"mayReadItems"`
	MayAddItems    bool `json:"mayAddItems"`
	MayRemoveItems bool `json:"mayRemoveItems"`
	MaySetSeen     bool `json:"maySetSeen"`
	MaySetKeywords bool `json:"maySetKeywords"`
	MayCreateChild bool `json:"mayCreateChild"`
	MayRename      bool `json:"mayRename"`
	MayDelete      bool `json:"mayDelete"`
	MaySubmit      bool `json:"maySubmit"`
}

// Helper functions for parsing JMAP responses

func getString(data map[string]interface{}, key string) string {
	if value, ok := data[key].(string); ok {
		return value
	}
	return ""
}

func getInt(data map[string]interface{}, key string) int {
	if value, ok := data[key].(float64); ok {
		return int(value)
	}
	if value, ok := data[key].(int); ok {
		return value
	}
	return 0
}

func getBool(data map[string]interface{}, key string) bool {
	if value, ok := data[key].(bool); ok {
		return value
	}
	return false
}
