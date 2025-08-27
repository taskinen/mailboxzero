package jmap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	endpoint   string
	apiToken   string
	httpClient *http.Client
	session    *Session
}

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

type Account struct {
	Name                string                 `json:"name"`
	IsPersonal          bool                   `json:"isPersonal"`
	IsReadOnly          bool                   `json:"isReadOnly"`
	AccountCapabilities map[string]interface{} `json:"accountCapabilities"`
}

type Request struct {
	Using  []string `json:"using"`
	Method string   `json:"methodCalls"`
	CallID string   `json:"callId,omitempty"`
}

type MethodCall []interface{}

type Response struct {
	MethodResponses [][]interface{} `json:"methodResponses"`
	SessionState    string          `json:"sessionState"`
}

func NewClient(endpoint, apiToken string) *Client {
	return &Client{
		endpoint: endpoint,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

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
	return nil
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

func (c *Client) GetPrimaryAccount() string {
	if c.session != nil && c.session.PrimaryAccounts != nil {
		if accountID, ok := c.session.PrimaryAccounts["urn:ietf:params:jmap:mail"]; ok {
			return accountID
		}
	}
	return ""
}
