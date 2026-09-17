// Package spotify provides a lightweight client-credentials Spotify client
// used for search, playlist lookups and track resolution in the assistant.
package spotify

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/vishalgojha/sdsheetal/internal/config"
)

type Client struct {
	cfg     *config.Config
	http    *http.Client
	mu      sync.Mutex
	token   string
	expires time.Time
}

type Track struct {
	URI    string `json:"uri"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Art    string `json:"art"`
	DurMS  int64  `json:"dur_ms"`
}

func New(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: time.Duration(cfg.SpotifyTimeoutS) * time.Second,
		},
	}
}

func (c *Client) Enabled() bool { return c.cfg.HasSpotify() }

func (c *Client) tokenValue() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expires) {
		return c.token, nil
	}
	if !c.cfg.HasSpotify() {
		return "", fmt.Errorf("Spotify credentials not configured")
	}
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	req, err := http.NewRequest(http.MethodPost, "https://accounts.spotify.com/api/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	basic := base64.StdEncoding.EncodeToString([]byte(c.cfg.SpotifyClientID + ":" + c.cfg.SpotifyClientSecret))
	req.Header.Set("Authorization", "Basic "+basic)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("spotify token error: %s", resp.Status)
	}
	var data struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	c.token = data.AccessToken
	c.expires = time.Now().Add(time.Duration(data.ExpiresIn-30) * time.Second)
	return c.token, nil
}

func (c *Client) get(path string, out any) error {
	token, err := c.tokenValue()
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodGet, "https://api.spotify.com/v1"+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("spotify API %s: %s", path, resp.Status)
	}
	return json.Unmarshal(body, out)
}

func parseTracks(raw []map[string]json.RawMessage) []Track {
	out := make([]Track, 0, len(raw))
	for _, item := range raw {
		var uri, id, name string
		json.Unmarshal(item["uri"], &uri)
		json.Unmarshal(item["id"], &id)
		json.Unmarshal(item["name"], &name)
		if uri == "" {
			continue
		}
		t := Track{URI: uri, ID: id, Name: name}
		json.Unmarshal(item["duration_ms"], &t.DurMS)

		var artists []struct {
			Name string `json:"name"`
		}
		json.Unmarshal(item["artists"], &artists)
		parts := make([]string, 0, len(artists))
		for _, a := range artists {
			parts = append(parts, a.Name)
		}
		t.Artist = strings.Join(parts, ", ")

		var album struct {
			Name   string `json:"name"`
			Images []struct {
				URL string `json:"url"`
			} `json:"images"`
		}
		json.Unmarshal(item["album"], &album)
		t.Album = album.Name
		if len(album.Images) > 0 {
			t.Art = album.Images[0].URL
		}
		out = append(out, t)
	}
	return out
}

// SearchTracks searches Spotify and returns the top matches.
func (c *Client) SearchTracks(q string, limit int) ([]Track, error) {
	if !c.Enabled() {
		return nil, nil
	}
	if limit <= 0 || limit > 20 {
		limit = 6
	}
	var data map[string]json.RawMessage
	err := c.get("/search?q="+url.QueryEscape(q)+"&type=track&limit="+fmt.Sprint(limit)+"&market=IN", &data)
	if err != nil {
		return nil, err
	}
	var tracks struct {
		Items []map[string]json.RawMessage `json:"items"`
	}
	_ = json.Unmarshal(data["tracks"], &tracks)
	return parseTracks(tracks.Items), nil
}

// PlaylistTracks returns the configured playlist's first tracks.
func (c *Client) PlaylistTracks(limit int) ([]Track, error) {
	if !c.Enabled() || c.cfg.SpotifyPlaylistID == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 30 {
		limit = 20
	}
	var data struct {
		Items []struct {
			Track map[string]json.RawMessage `json:"track"`
		} `json:"items"`
	}
	err := c.get("/playlists/"+url.PathEscape(c.cfg.SpotifyPlaylistID)+"/items?limit="+fmt.Sprint(limit)+"&market=IN", &data)
	if err != nil {
		return nil, err
	}
	raw := make([]map[string]json.RawMessage, 0, len(data.Items))
	for _, item := range data.Items {
		if item.Track != nil {
			raw = append(raw, item.Track)
		}
	}
	return parseTracks(raw), nil
}

// FindTrack resolves a full track object for a spotify:track URI.
func (c *Client) FindTrack(uri string) (*Track, error) {
	if !c.Enabled() {
		return nil, nil
	}
	parts := strings.Split(uri, ":")
	if len(parts) < 3 {
		return nil, fmt.Errorf("not a spotify URI")
	}
	var data map[string]json.RawMessage
	if err := c.get("/tracks/"+url.PathEscape(parts[len(parts)-1]), &data); err != nil {
		return nil, err
	}
	parsed := parseTracks([]map[string]json.RawMessage{data})
	if len(parsed) == 0 {
		return nil, fmt.Errorf("track not found")
	}
	return &parsed[0], nil
}

// ToJSON returns a plain JSON-friendly map for API responses.
func (t Track) ToJSON() map[string]any {
	return map[string]any{
		"uri": t.URI, "id": t.ID, "name": t.Name, "artist": t.Artist,
		"album": t.Album, "art": t.Art, "dur_ms": t.DurMS,
	}
}