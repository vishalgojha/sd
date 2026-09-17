// Package gmail provides read-only Gmail access through Nango's provider
// proxy, matching the original app's integration.
package gmail

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vishalgojha/sdsheetal/internal/config"
)

type Client struct {
	cfg        *config.Config
	http       *http.Client
	connection *Connection
}

type Connection struct {
	ConnectionID string `json:"connection_id"`
	Provider     string `json:"provider"`
	UserID       string `json:"user_id"`
	UpdatedAt    string `json:"updated_at"`
}

type Message struct {
	ID       string `json:"id"`
	ThreadID string `json:"thread_id"`
	From     string `json:"from"`
	Subject  string `json:"subject"`
	Date     string `json:"date"`
	Snippet  string `json:"snippet"`
	Unread   bool   `json:"unread"`
}

func New(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: 18 * time.Second,
		},
	}
}

func (c *Client) Enabled() bool { return c.cfg.HasNango() }

func (c *Client) SetConnection(conn *Connection) { c.connection = conn }

func (c *Client) Connection() *Connection { return c.connection }

func (c *Client) nangoRequest(method, path string, payload any, headers map[string]string) ([]byte, error) {
	if c.cfg.NangoSecretKey == "" {
		return nil, fmt.Errorf("Nango is not configured")
	}
	var raw []byte
	if payload != nil {
		raw, _ = json.Marshal(payload)
	}
	req, err := http.NewRequest(method, strings.TrimRight(c.cfg.NangoAPIBase, "/")+"/"+strings.TrimLeft(path, "/"), bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.NangoSecretKey)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("nango %s: %s", resp.Status, snippet(string(body)))
	}
	return body, nil
}

func snippet(s string) string {
	runes := []rune(s)
	if len(runes) > 220 {
		return string(runes[:220])
	}
	return s
}

// ConnectLink creates a hosted Nango Connect session for Gmail.
func (c *Client) ConnectLink() (string, error) {
	body, err := c.nangoRequest("POST", "/connect/sessions", map[string]any{
		"tags": map[string]string{
			"end_user_id":        c.cfg.NangoUserID,
			"end_user_display_name": "Sheetal",
		},
		"allowed_integrations": []string{c.cfg.NangoIntegration},
	}, nil)
	if err != nil {
		return "", err
	}
	var parsed struct {
		Data struct {
			ConnectLink string `json:"connect_link"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.Data.ConnectLink == "" {
		return "", fmt.Errorf("Nango did not return a connect link")
	}
	return parsed.Data.ConnectLink, nil
}

// RefreshConnection resolves the stored or first Gmail connection for the user.
func (c *Client) RefreshConnection() (string, error) {
	if c.connection != nil && c.connection.ConnectionID != "" {
		return c.connection.ConnectionID, nil
	}
	if c.cfg.NangoIntegration == "" {
		return "", fmt.Errorf("Nango integration not configured")
	}
	params := url.Values{}
	params.Set("tags[end_user_id]", c.cfg.NangoUserID)
	params.Set("limit", "20")
	body, err := c.nangoRequest("GET", "/connections?"+params.Encode(), nil, nil)
	if err != nil {
		return "", err
	}
	var parsed struct {
		Connections []struct {
			Provider_config_key string `json:"provider_config_key"`
			ProviderConfigKey   string `json:"providerConfigKey"`
			Connection_id       string `json:"connection_id"`
			ConnectionId        string `json:"connectionId"`
		} `json:"connections"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	for _, conn := range parsed.Connections {
		provider := firstNonEmpty(conn.Provider_config_key, conn.ProviderConfigKey)
		id := firstNonEmpty(conn.Connection_id, conn.ConnectionId)
		if provider == c.cfg.NangoIntegration && id != "" {
			c.connection = &Connection{
				ConnectionID: id, Provider: provider, UserID: c.cfg.NangoUserID,
				UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			}
			return id, nil
		}
	}
	return "", fmt.Errorf("no Gmail connection found for this user")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// Messages reads up to limit Gmail messages with metadata only.
func (c *Client) Messages(query string, limit int) ([]Message, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	connID, err := c.RefreshConnection()
	if err != nil {
		return nil, err
	}
	headers := map[string]string{
		"Provider-Config-Key": c.cfg.NangoIntegration,
		"Connection-Id":       connID,
	}
	params := url.Values{}
	params.Set("maxResults", fmt.Sprint(limit))
	if q := strings.TrimSpace(query); q != "" {
		params.Set("q", q)
	}
	body, err := c.nangoRequest("GET", "/proxy/gmail/v1/users/me/messages?"+params.Encode(), nil, headers)
	if err != nil {
		return nil, err
	}
	var list struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, err
	}
	out := make([]Message, 0, len(list.Messages))
	for i, item := range list.Messages {
		if i >= limit {
			break
		}
		detail, err := c.nangoRequest("GET", "/proxy/gmail/v1/users/me/messages/"+url.PathEscape(item.ID)+"?"+url.Values{
			"format":           {"metadata"},
			"metadataHeaders":  {"From", "Subject", "Date"},
		}.Encode(), nil, headers)
		if err != nil {
			continue
		}
		var message struct {
			ID       string   `json:"id"`
			ThreadID string   `json:"threadId"`
			Snippet  string   `json:"snippet"`
			LabelIDs []string `json:"labelIds"`
			Payload  struct {
				Headers []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"headers"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(detail, &message); err != nil {
			continue
		}
		headersMap := map[string]string{}
		for _, h := range message.Payload.Headers {
			headersMap[strings.ToLower(h.Name)] = h.Value
		}
		unread := false
		for _, l := range message.LabelIDs {
			if l == "UNREAD" {
				unread = true
				break
			}
		}
		subject := strings.TrimSpace(headersMap["subject"])
		if subject == "" {
			subject = "(no subject)"
		}
		out = append(out, Message{
			ID: message.ID, ThreadID: message.ThreadID,
			From: strings.TrimSpace(headersMap["from"]), Subject: subject,
			Date: strings.TrimSpace(headersMap["date"]), Snippet: message.Snippet,
			Unread: unread,
		})
	}
	return out, nil
}