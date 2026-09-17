// Package sarvam wraps the Sarvam AI REST API (api.sarvam.ai) for Indic
// language speech: text-to-speech (Bulbul models) and speech-to-text
// (Saaras models, incl. translate/translit/verbatim/codemix modes).
package sarvam

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/vishalgojha/sdsheetal/internal/config"
)

const defaultBase = "https://api.sarvam.ai"

// Client is a thin Sarvam AI REST client.
type Client struct {
	cfg      *config.Config
	base     string
	apiKey   string
	http     *http.Client
}

func New(cfg *config.Config) *Client {
	return &Client{
		cfg:    cfg,
		base:   defaultBase,
		apiKey: cfg.SarvamAPIKey,
		http: &http.Client{
			Timeout: 40 * time.Second,
		},
	}
}

// Enabled reports whether an API key was configured.
func (c *Client) Enabled() bool { return c.apiKey != "" }

// Speak renders text to audio bytes (MP3 when the request asks for it).
func (c *Client) Speak(text string) ([]byte, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("Sarvam not configured")
	}
	body, err := json.Marshal(map[string]any{
		"text":               text,
		"language_code":      c.cfg.SarvamTTSLang,
		"speaker":            c.cfg.SarvamTTSSpeaker,
		"model":              c.cfg.SarvamTTSModel,
		"speech_sample_rate": 24000,
		"output_audio_codec": "mp3",
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.base+"/text-to-speech", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("api-subscription-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sarvam TTS %s: %s", resp.Status, truncate(string(raw), 160))
	}
	var out struct {
		Audios []string `json:"audios"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("sarvam TTS: bad response: %w", err)
	}
	if len(out.Audios) == 0 || strings.TrimSpace(out.Audios[0]) == "" {
		return nil, fmt.Errorf("sarvam TTS: empty audio response")
	}
	return base64.StdEncoding.DecodeString(out.Audios[0])
}

// Transcribe converts an audio sample to text. fileName is only used to hint
// the upload's filename; mime provides the content type. Language defaults to
// the configured code (typically "hi-IN"); pass "unknown" for auto-detection.
func (c *Client) Transcribe(ctx context.Context, fileName string, mime string, audio []byte) (string, error) {
	if !c.Enabled() {
		return "", fmt.Errorf("Sarvam not configured")
	}
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	if fileName == "" {
		fileName = defaultFileName(mime)
	}
	fw, err := mw.CreateFormFile("file", path.Base(fileName))
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(audio); err != nil {
		return "", err
	}
	for _, field := range []struct{ name, value string }{
		{"model", c.cfg.SarvamSTTModel},
		{"mode", c.cfg.SarvamSTTMode},
		{"language_code", c.cfg.SarvamSTTLang},
	} {
		if field.value == "" {
			continue
		}
		_ = mw.WriteField(field.name, field.value)
	}
	if err := mw.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/speech-to-text", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("api-subscription-key", c.apiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("sarvam STT %s: %s", resp.Status, truncate(string(raw), 160))
	}
	var out struct {
		Transcript   string `json:"transcript"`
		LanguageCode string `json:"language_code"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("sarvam STT: bad response: %w", err)
	}
	return strings.TrimSpace(out.Transcript), nil
}

// defaultFileName maps a mimetype to a sensible upload filename.
func defaultFileName(mime string) string {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "audio/ogg", "audio/ogg; codecs=opus", "audio/opus":
		return "voice.ogg"
	case "audio/mpeg", "audio/mp3":
		return "voice.mp3"
	case "audio/wav", "audio/wave":
		return "voice.wav"
	case "audio/mp4", "audio/m4a":
		return "voice.m4a"
	case "audio/amr":
		return "voice.amr"
	default:
		return "voice.audio"
	}
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n])
	}
	return s
}