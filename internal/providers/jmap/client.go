// Package jmap provides JMAP (JSON Meta Application Protocol) implementation
package jmap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mailboxzero/internal/protocol"
	"net/http"
	"time"
)

// Client implements protocol.EmailClient for JMAP
type Client struct {
	endpoint   string
	apiToken   string
	httpClient *http.Client
	session    *Session

	// Cached mailbox IDs for performance
	inboxID   string
	archiveID string
}

// NewClient creates a new JMAP client
func NewClient(endpoint, apiToken string) *Client {
	return &Client{
		endpoint: endpoint,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Authenticate implements protocol.EmailClient
func (c *Client) Authenticate() error {
	req, err := http.NewRequest("GET", c.endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create session request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed: %d - %s", resp.StatusCode, string(body))
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return fmt.Errorf("failed to decode session: %w", err)
	}

	c.session = &session

	// Cache mailbox IDs during authentication for performance
	return c.cacheMailboxIDs()
}

// GetInboxEmailsWithCountPaginated implements protocol.EmailClient
func (c *Client) GetInboxEmailsWithCountPaginated(limit, offset int) (*protocol.InboxInfo, error) {
	accountID := c.getPrimaryAccount()
	if accountID == "" {
		return nil, fmt.Errorf("no primary account found")
	}

	// Get total count from cached mailbox info
	mailboxes, err := c.getMailboxes()
	if err != nil {
		return nil, fmt.Errorf("failed to get mailboxes: %w", err)
	}

	var totalCount int
	for _, mb := range mailboxes {
		if mb.Role == "inbox" {
			totalCount = mb.TotalEmails
			break
		}
	}

	// Get emails using paginated query
	jmapEmails, err := c.getInboxEmailsPaginated(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get inbox emails: %w", err)
	}

	// Convert JMAP emails to protocol emails
	emails := make([]protocol.Email, len(jmapEmails))
	for i, je := range jmapEmails {
		emails[i] = convertJMAPToProtocolEmail(je)
	}

	return &protocol.InboxInfo{
		Emails:     emails,
		TotalCount: totalCount,
	}, nil
}

// GetInboxEmails implements protocol.EmailClient
func (c *Client) GetInboxEmails(limit int) ([]protocol.Email, error) {
	info, err := c.GetInboxEmailsWithCountPaginated(limit, 0)
	if err != nil {
		return nil, err
	}
	return info.Emails, nil
}

// ArchiveEmails implements protocol.EmailClient
func (c *Client) ArchiveEmails(emailIDs []string, dryRun bool) error {
	if dryRun {
		fmt.Printf("[JMAP DRY RUN] Would archive %d emails: %v\n", len(emailIDs), emailIDs)
		return nil
	}

	accountID := c.getPrimaryAccount()
	if accountID == "" {
		return fmt.Errorf("no primary account found")
	}

	// Use cached mailbox IDs
	if c.inboxID == "" || c.archiveID == "" {
		return fmt.Errorf("inbox or archive folder not found")
	}

	updates := make(map[string]interface{})
	for _, emailID := range emailIDs {
		updates[emailID] = map[string]interface{}{
			"mailboxIds": map[string]bool{
				c.archiveID: true,
			},
		}
	}

	methodCalls := []MethodCall{
		{"Email/set", map[string]interface{}{
			"accountId": accountID,
			"update":    updates,
		}, "0"},
	}

	_, err := c.makeRequest(methodCalls)
	if err != nil {
		return fmt.Errorf("failed to archive emails: %w", err)
	}

	return nil
}

// Close implements protocol.EmailClient
func (c *Client) Close() error {
	// JMAP doesn't require explicit connection closing
	return nil
}

// Internal methods

func (c *Client) cacheMailboxIDs() error {
	mailboxes, err := c.getMailboxes()
	if err != nil {
		return fmt.Errorf("failed to cache mailbox IDs: %w", err)
	}

	for _, mb := range mailboxes {
		if mb.Role == "inbox" {
			c.inboxID = mb.ID
		}
		if mb.Role == "archive" {
			c.archiveID = mb.ID
		}
	}

	if c.inboxID == "" {
		return fmt.Errorf("inbox not found")
	}
	if c.archiveID == "" {
		return fmt.Errorf("archive folder not found")
	}

	return nil
}

func (c *Client) getPrimaryAccount() string {
	if c.session != nil && c.session.PrimaryAccounts != nil {
		if accountID, ok := c.session.PrimaryAccounts["urn:ietf:params:jmap:mail"]; ok {
			return accountID
		}
	}
	return ""
}

func (c *Client) makeRequest(methodCalls []MethodCall) (*Response, error) {
	if c.session == nil {
		return nil, fmt.Errorf("client not authenticated")
	}

	reqBody := map[string]interface{}{
		"using":       []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		"methodCalls": methodCalls,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.session.APIUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %d - %s", resp.StatusCode, string(body))
	}

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

func (c *Client) getMailboxes() ([]Mailbox, error) {
	accountID := c.getPrimaryAccount()
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

func (c *Client) getInboxEmailsPaginated(limit, offset int) ([]jmapEmail, error) {
	accountID := c.getPrimaryAccount()
	if accountID == "" {
		return nil, fmt.Errorf("no primary account found")
	}

	// Use cached inbox ID
	if c.inboxID == "" {
		return nil, fmt.Errorf("inbox not found")
	}

	queryParams := map[string]interface{}{
		"accountId": accountID,
		"filter": map[string]interface{}{
			"inMailbox": c.inboxID,
		},
		"sort": []map[string]interface{}{
			{"property": "receivedAt", "isAscending": false},
		},
		"limit": limit,
	}

	if offset > 0 {
		queryParams["position"] = offset
	}

	methodCalls := []MethodCall{
		{"Email/query", queryParams, "0"},
		{"Email/get", map[string]interface{}{
			"accountId": accountID,
			"#ids":      map[string]interface{}{"resultOf": "0", "name": "Email/query", "path": "/ids"},
			"properties": []string{
				"id", "subject", "from", "to", "receivedAt", "preview", "hasAttachment", "mailboxIds", "keywords",
				"bodyValues", "textBody", "htmlBody",
			},
			"bodyProperties":      []string{"value", "isEncodingProblem", "isTruncated"},
			"fetchTextBodyValues": true,
			"fetchHTMLBodyValues": true,
			"maxBodyValueBytes":   50000,
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

	var emails []jmapEmail
	for _, item := range emailsData {
		emailData, _ := item.(map[string]interface{})
		email := parseJMAPEmail(emailData)
		emails = append(emails, email)
	}

	return emails, nil
}

// Conversion functions

func convertJMAPToProtocolEmail(je jmapEmail) protocol.Email {
	email := protocol.Email{
		ID:            je.ID,
		Subject:       je.Subject,
		Preview:       je.Preview,
		ReceivedAt:    je.ReceivedAt,
		SentAt:        je.SentAt,
		HasAttachment: je.HasAttachment,
		Size:          je.Size,
		From:          convertAddresses(je.From),
		To:            convertAddresses(je.To),
		Cc:            convertAddresses(je.Cc),
		Bcc:           convertAddresses(je.Bcc),
		ReplyTo:       convertAddresses(je.ReplyTo),
	}

	// Extract body text from JMAP's complex structure
	email.BodyText = extractTextBody(je)
	email.BodyHTML = extractHTMLBody(je)

	// Convert JMAP keywords to generic flags
	for keyword := range je.Keywords {
		email.Flags = append(email.Flags, keyword)
	}

	// Store JMAP-specific metadata
	email.Metadata = map[string]interface{}{
		"threadId": je.ThreadID,
		"blobId":   je.BlobID,
	}

	return email
}

func convertAddresses(jmapAddrs []jmapEmailAddress) []protocol.EmailAddress {
	addrs := make([]protocol.EmailAddress, len(jmapAddrs))
	for i, ja := range jmapAddrs {
		addrs[i] = protocol.EmailAddress{
			Name:  ja.Name,
			Email: ja.Email,
		}
	}
	return addrs
}

func extractTextBody(je jmapEmail) string {
	// Try to get text body from BodyValues
	for _, part := range je.TextBody {
		if bodyValue, ok := je.BodyValues[part.PartID]; ok {
			if bodyValue.Value != "" {
				return bodyValue.Value
			}
		}
	}

	// Fallback to any body value
	for _, bodyValue := range je.BodyValues {
		if bodyValue.Value != "" {
			return bodyValue.Value
		}
	}

	return ""
}

func extractHTMLBody(je jmapEmail) string {
	// Try to get HTML body from BodyValues
	for _, part := range je.HTMLBody {
		if bodyValue, ok := je.BodyValues[part.PartID]; ok {
			if bodyValue.Value != "" {
				return bodyValue.Value
			}
		}
	}

	return ""
}

func parseJMAPEmail(data map[string]interface{}) jmapEmail {
	email := jmapEmail{
		ID:      getString(data, "id"),
		Subject: getString(data, "subject"),
		Preview: getString(data, "preview"),
	}

	if receivedAtStr := getString(data, "receivedAt"); receivedAtStr != "" {
		if t, err := time.Parse(time.RFC3339, receivedAtStr); err == nil {
			email.ReceivedAt = t
		}
	}

	if sentAtStr := getString(data, "sentAt"); sentAtStr != "" {
		if t, err := time.Parse(time.RFC3339, sentAtStr); err == nil {
			email.SentAt = t
		}
	}

	if fromData, ok := data["from"].([]interface{}); ok && len(fromData) > 0 {
		if fromMap, ok := fromData[0].(map[string]interface{}); ok {
			email.From = []jmapEmailAddress{{
				Name:  getString(fromMap, "name"),
				Email: getString(fromMap, "email"),
			}}
		}
	}

	// Parse textBody structure
	if textBodyData, ok := data["textBody"].([]interface{}); ok {
		for _, part := range textBodyData {
			if partMap, ok := part.(map[string]interface{}); ok {
				email.TextBody = append(email.TextBody, BodyPart{
					PartID: getString(partMap, "partId"),
					Type:   getString(partMap, "type"),
				})
			}
		}
	}

	// Parse htmlBody structure
	if htmlBodyData, ok := data["htmlBody"].([]interface{}); ok {
		for _, part := range htmlBodyData {
			if partMap, ok := part.(map[string]interface{}); ok {
				email.HTMLBody = append(email.HTMLBody, BodyPart{
					PartID: getString(partMap, "partId"),
					Type:   getString(partMap, "type"),
				})
			}
		}
	}

	// Parse bodyValues
	if bodyValues, ok := data["bodyValues"].(map[string]interface{}); ok {
		email.BodyValues = make(map[string]BodyValue)
		for key, value := range bodyValues {
			if bodyMap, ok := value.(map[string]interface{}); ok {
				email.BodyValues[key] = BodyValue{
					Value:             getString(bodyMap, "value"),
					IsEncodingProblem: getBool(bodyMap, "isEncodingProblem"),
					IsTruncated:       getBool(bodyMap, "isTruncated"),
				}
			}
		}
	}

	// Parse keywords
	if keywords, ok := data["keywords"].(map[string]interface{}); ok {
		email.Keywords = make(map[string]bool)
		for key, value := range keywords {
			if boolVal, ok := value.(bool); ok {
				email.Keywords[key] = boolVal
			}
		}
	}

	email.HasAttachment = getBool(data, "hasAttachment")
	email.Size = getInt(data, "size")
	email.ThreadID = getString(data, "threadId")
	email.BlobID = getString(data, "blobId")

	return email
}
