// Package eleven provides a small bridge to an ElevenLabs Conversational AI
// agent. The agent owns the natural-language conversation; Sheetal's local
// tools remain available through the deterministic command path and can be
// connected to ElevenLabs client tools later without changing this API.
package eleven

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/vishalgojha/sdsheetal/internal/config"
)

type Client struct {
	cfg  *config.Config
	http *http.Client
}

func New(cfg *config.Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: 20 * time.Second}}
}

// Enabled is true only when an agent ID and API key are both present.
func (c *Client) Enabled() bool {
	return c != nil && c.cfg != nil && c.cfg.ElevenLabsAPIKey != "" && c.cfg.ElevenLabsAgentID != ""
}

func (c *Client) signedURL(ctx context.Context) (string, error) {
	u := "https://api.elevenlabs.io/v1/convai/conversation/get-signed-url?agent_id=" + url.QueryEscape(c.cfg.ElevenLabsAgentID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("xi-api-key", c.cfg.ElevenLabsAPIKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("elevenlabs signed URL %s: %s", resp.Status, truncate(string(raw), 180))
	}
	var out struct {
		SignedURL string `json:"signed_url"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.SignedURL == "" {
		return "", fmt.Errorf("elevenlabs: signed URL missing")
	}
	return out.SignedURL, nil
}

// Chat sends one turn through the configured ElevenLabs Conversational AI
// agent and returns its completed response text.
func (c *Client) Chat(ctx context.Context, system, user string) (string, error) {
	if !c.Enabled() {
		return "", fmt.Errorf("ElevenLabs agent not configured")
	}
	signed, err := c.signedURL(ctx)
	if err != nil {
		return "", err
	}
	conn, _, err := websocket.Dial(ctx, signed, &websocket.DialOptions{HTTPClient: c.http})
	if err != nil {
		return "", err
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")
	init := map[string]any{"type": "conversation_initiation_client_data", "conversation_config_override": map[string]any{}, "dynamic_variables": map[string]string{"device_context": "WhatsApp self-chat; India; IST", "assistant_context": system}}
	if err := writeJSON(ctx, conn, init); err != nil {
		return "", err
	}
	var answer strings.Builder
	sent := false
	deadline := time.NewTimer(45 * time.Second)
	defer deadline.Stop()
	for {
		readCtx, cancel := context.WithCancel(ctx)
		go func() {
			select {
			case <-deadline.C:
				cancel()
			case <-readCtx.Done():
			}
		}()
		t, data, err := conn.Read(readCtx)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			return "", err
		}
		if t != websocket.MessageText {
			continue
		}
		var event map[string]any
		if json.Unmarshal(data, &event) != nil {
			continue
		}
		typ, _ := event["type"].(string)
		switch typ {
		case "conversation_initiation_metadata":
			if !sent {
				if err := writeJSON(ctx, conn, map[string]any{"type": "user_message", "text": user}); err != nil {
					return "", err
				}
				sent = true
			}
		case "ping":
			ping := event["ping_event"]
			id := event["event_id"]
			if p, ok := ping.(map[string]any); ok {
				id = p["event_id"]
			}
			_ = writeJSON(ctx, conn, map[string]any{"type": "pong", "event_id": id})
		case "client_error":
			return "", fmt.Errorf("elevenlabs agent error: %s", firstString(event, "message", "error"))
		case "agent_response", "agent_chat_response_part", "agent_response_correction":
			if s := eventText(event); s != "" {
				answer.WriteString(s)
			}
		case "agent_response_complete":
			if s := strings.TrimSpace(answer.String()); s != "" {
				return s, nil
			}
		}
	}
}

func writeJSON(ctx context.Context, c *websocket.Conn, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Write(ctx, websocket.MessageText, b)
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func eventText(v any) string {
	if m, ok := v.(map[string]any); ok {
		for _, k := range []string{"text", "content", "agent_response", "text_response_part", "agent_response_event"} {
			if x, exists := m[k]; exists {
				if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
					return s
				}
				if s := eventText(x); s != "" {
					return s
				}
			}
		}
		for k, x := range m {
			if k != "type" && k != "event_id" {
				if s := eventText(x); s != "" {
					return s
				}
			}
		}
	}
	if xs, ok := v.([]any); ok {
		for _, x := range xs {
			if s := eventText(x); s != "" {
				return s
			}
		}
	}
	return ""
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
