// Package tts wraps the ElevenLabs text-to-speech REST API so the assistant
// can reply to WhatsApp messages with voice notes.
package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/vishalgojha/sdsheetal/internal/config"
)

type Client struct {
	cfg  *config.Config
	http *http.Client
}

func New(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Enabled() bool { return c.cfg.HasElevenLabs() }

// Speak renders the text to MP3 audio bytes via the tuned voice settings
// used by the original app.
func (c *Client) Speak(text string) ([]byte, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("ElevenLabs not configured")
	}
	body, err := json.Marshal(map[string]any{
		"text": text,
		"model_id": c.cfg.ElevenLabsModel,
		"voice_settings": map[string]any{
			"stability":         0.45,
			"similarity_boost":  0.8,
			"style":             0.6,
		},
	})
	if err != nil {
		return nil, err
	}
	endpoint := "https://api.elevenlabs.io/v1/text-to-speech/" + url.PathEscape(c.cfg.ElevenLabsVoice)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("xi-api-key", c.cfg.ElevenLabsAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("elevenlabs %s: %s", resp.Status, truncate(string(audio), 120))
	}
	return audio, nil
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n])
	}
	return s
}