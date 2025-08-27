package jmap

import (
	"fmt"
	"time"
)

type Email struct {
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
	Sender        []EmailAddress       `json:"sender"`
	From          []EmailAddress       `json:"from"`
	To            []EmailAddress       `json:"to"`
	Cc            []EmailAddress       `json:"cc"`
	Bcc           []EmailAddress       `json:"bcc"`
	ReplyTo       []EmailAddress       `json:"replyTo"`
	Subject       string               `json:"subject"`
	SentAt        time.Time            `json:"sentAt"`
	HasAttachment bool                 `json:"hasAttachment"`
	Preview       string               `json:"preview"`
	BodyValues    map[string]BodyValue `json:"bodyValues"`
	TextBody      []BodyPart           `json:"textBody"`
	HTMLBody      []BodyPart           `json:"htmlBody"`
	Attachments   []Attachment         `json:"attachments"`
}

type EmailAddress struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type BodyValue struct {
	Value             string `json:"value"`
	IsEncodingProblem bool   `json:"isEncodingProblem"`
	IsTruncated       bool   `json:"isTruncated"`
}

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

func (c *Client) GetMailboxes() ([]Mailbox, error) {
	accountID := c.GetPrimaryAccount()
	if accountID == "" {
		return nil, fmt.Errorf("no primary account found")
	}

	methodCalls := []MethodCall{
		{"Mailbox/get", map[string]interface{}{
			"accountId": accountID,
		}, "0"},
	}

	resp, err := c.makeRequest(methodCalls)
	if err != nil {
		return nil, fmt.Errorf("failed to get mailboxes: %w", err)
	}

	if len(resp.MethodResponses) == 0 {
		return nil, fmt.Errorf("no response received")
	}

	response := resp.MethodResponses[0]
	if len(response) < 2 {
		return nil, fmt.Errorf("invalid response format")
	}

	responseData, ok := response[1].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response data format")
	}

	mailboxesData, ok := responseData["list"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid mailboxes data format")
	}

	var mailboxes []Mailbox
	for _, item := range mailboxesData {
		mailboxData, _ := item.(map[string]interface{})
		mailbox := Mailbox{
			ID:           getString(mailboxData, "id"),
			Name:         getString(mailboxData, "name"),
			Role:         getString(mailboxData, "role"),
			TotalEmails:  getInt(mailboxData, "totalEmails"),
			UnreadEmails: getInt(mailboxData, "unreadEmails"),
		}
		mailboxes = append(mailboxes, mailbox)
	}

	return mailboxes, nil
}

func (c *Client) GetInboxEmails(limit int) ([]Email, error) {
	accountID := c.GetPrimaryAccount()
	if accountID == "" {
		return nil, fmt.Errorf("no primary account found")
	}

	mailboxes, err := c.GetMailboxes()
	if err != nil {
		return nil, fmt.Errorf("failed to get mailboxes: %w", err)
	}

	var inboxID string
	for _, mb := range mailboxes {
		if mb.Role == "inbox" {
			inboxID = mb.ID
			break
		}
	}

	if inboxID == "" {
		return nil, fmt.Errorf("inbox not found")
	}

	methodCalls := []MethodCall{
		{"Email/query", map[string]interface{}{
			"accountId": accountID,
			"filter": map[string]interface{}{
				"inMailbox": inboxID,
			},
			"sort": []map[string]interface{}{
				{"property": "receivedAt", "isAscending": false},
			},
			"limit": limit,
		}, "0"},
		{"Email/get", map[string]interface{}{
			"accountId": accountID,
			"#ids":      map[string]interface{}{"resultOf": "0", "name": "Email/query", "path": "/ids"},
			"properties": []string{
				"id", "subject", "from", "to", "receivedAt", "preview", "hasAttachment", "mailboxIds", "keywords",
				"bodyValues", "textBody", "htmlBody",
			},
			"bodyProperties": []string{"value", "isEncodingProblem", "isTruncated"},
		}, "1"},
	}

	resp, err := c.makeRequest(methodCalls)
	if err != nil {
		return nil, fmt.Errorf("failed to get emails: %w", err)
	}

	if len(resp.MethodResponses) < 2 {
		return nil, fmt.Errorf("insufficient responses received")
	}

	emailGetResponse := resp.MethodResponses[1]
	if len(emailGetResponse) < 2 {
		return nil, fmt.Errorf("invalid email get response format")
	}

	responseData, ok := emailGetResponse[1].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response data format")
	}

	emailsData, ok := responseData["list"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid emails data format")
	}

	var emails []Email
	for _, item := range emailsData {
		emailData, _ := item.(map[string]interface{})
		email := parseEmail(emailData)
		emails = append(emails, email)
	}

	return emails, nil
}

type InboxInfo struct {
	Emails     []Email `json:"emails"`
	TotalCount int     `json:"totalCount"`
}

func (c *Client) GetInboxEmailsWithCount(limit int) (*InboxInfo, error) {
	accountID := c.GetPrimaryAccount()
	if accountID == "" {
		return nil, fmt.Errorf("no primary account found")
	}

	mailboxes, err := c.GetMailboxes()
	if err != nil {
		return nil, fmt.Errorf("failed to get mailboxes: %w", err)
	}

	var inboxID string
	var totalCount int
	for _, mb := range mailboxes {
		if mb.Role == "inbox" {
			inboxID = mb.ID
			totalCount = mb.TotalEmails
			break
		}
	}

	if inboxID == "" {
		return nil, fmt.Errorf("inbox not found")
	}

	emails, err := c.GetInboxEmails(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get inbox emails: %w", err)
	}

	return &InboxInfo{
		Emails:     emails,
		TotalCount: totalCount,
	}, nil
}

func (c *Client) ArchiveEmails(emailIDs []string, dryRun bool) error {
	if dryRun {
		fmt.Printf("[DRY RUN] Would archive %d emails: %v\n", len(emailIDs), emailIDs)
		return nil
	}

	accountID := c.GetPrimaryAccount()
	if accountID == "" {
		return fmt.Errorf("no primary account found")
	}

	mailboxes, err := c.GetMailboxes()
	if err != nil {
		return fmt.Errorf("failed to get mailboxes: %w", err)
	}

	var inboxID, archiveID string
	for _, mb := range mailboxes {
		if mb.Role == "inbox" {
			inboxID = mb.ID
		}
		if mb.Role == "archive" {
			archiveID = mb.ID
		}
	}

	if inboxID == "" {
		return fmt.Errorf("inbox not found")
	}
	if archiveID == "" {
		return fmt.Errorf("archive folder not found")
	}

	updates := make(map[string]interface{})
	for _, emailID := range emailIDs {
		updates[emailID] = map[string]interface{}{
			"mailboxIds": map[string]bool{
				archiveID: true,
			},
		}
	}

	methodCalls := []MethodCall{
		{"Email/set", map[string]interface{}{
			"accountId": accountID,
			"update":    updates,
		}, "0"},
	}

	_, err = c.makeRequest(methodCalls)
	if err != nil {
		return fmt.Errorf("failed to archive emails: %w", err)
	}

	return nil
}

func parseEmail(data map[string]interface{}) Email {
	email := Email{
		ID:      getString(data, "id"),
		Subject: getString(data, "subject"),
		Preview: getString(data, "preview"),
	}

	if receivedAtStr := getString(data, "receivedAt"); receivedAtStr != "" {
		if t, err := time.Parse(time.RFC3339, receivedAtStr); err == nil {
			email.ReceivedAt = t
		}
	}

	if fromData, ok := data["from"].([]interface{}); ok && len(fromData) > 0 {
		if fromMap, ok := fromData[0].(map[string]interface{}); ok {
			email.From = []EmailAddress{{
				Name:  getString(fromMap, "name"),
				Email: getString(fromMap, "email"),
			}}
		}
	}

	if bodyValues, ok := data["bodyValues"].(map[string]interface{}); ok {
		email.BodyValues = make(map[string]BodyValue)
		for key, value := range bodyValues {
			if bodyMap, ok := value.(map[string]interface{}); ok {
				email.BodyValues[key] = BodyValue{
					Value: getString(bodyMap, "value"),
				}
			}
		}
	}

	return email
}

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
