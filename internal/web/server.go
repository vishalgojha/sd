// Package web serves the pairing / monitoring UI and the JSON API that backs
// it, plus a command console that drives the same agent as WhatsApp.
package web

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vishalgojha/sdsheetal/internal/assistant"
	"github.com/vishalgojha/sdsheetal/internal/config"
	"github.com/vishalgojha/sdsheetal/internal/gmail"
	"github.com/vishalgojha/sdsheetal/internal/spotify"
	"github.com/vishalgojha/sdsheetal/internal/store"
	"github.com/vishalgojha/sdsheetal/internal/whatsapp"
)

//go:embed static
var staticFiles embed.FS

type Server struct {
	cfg  *config.Config
	store *store.Store
	agent *assistant.Agent
	spot  *spotify.Client
	gmail *gmail.Client
	wa    *whatsapp.Client
	mux   *http.ServeMux
}

func New(cfg *config.Config, st *store.Store, agent *assistant.Agent, sp *spotify.Client, gm *gmail.Client, wa *whatsapp.Client) *Server {
	s := &Server{cfg: cfg, store: st, agent: agent, spot: sp, gmail: gm, wa: wa}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	mux := http.NewServeMux()
	s.mux = mux

	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			serveStatic(w, r, sub, "index.html", "text/html; charset=utf-8")
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	// WhatsApp pairing / status
	mux.HandleFunc("/api/whatsapp/status", s.status)
	mux.HandleFunc("/api/whatsapp/qr.png", s.qrPNG)
	mux.HandleFunc("/api/command", s.command)

	// Dashboard + tools
	mux.HandleFunc("/api/dashboard", s.dashboard)
	mux.HandleFunc("/api/status", s.radioStatus)
	mux.HandleFunc("/api/search", s.search)
	mux.HandleFunc("/api/queue", s.queue)
	mux.HandleFunc("/api/queue/next", s.queueNext)
	mux.HandleFunc("/api/queue/remove", s.queueRemove)

	mux.HandleFunc("/api/email/status", s.emailStatus)
	mux.HandleFunc("/api/email/connect", s.emailConnect)
	mux.HandleFunc("/api/email/inbox", s.emailInbox)

	mux.HandleFunc("/api/music/state", s.musicState)
	mux.HandleFunc("/api/music/command", s.musicCommand)

	mux.HandleFunc("/api/memory/event", s.memoryEvent)
}

func (s *Server) authed(r *http.Request) bool {
	if s.cfg.AgentToken == "" {
		return true
	}
	return r.Header.Get("X-SD-Agent-Token") == s.cfg.AgentToken ||
		r.Header.Get("X-SD-Token") == s.cfg.AgentToken
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func serveStatic(w http.ResponseWriter, r *http.Request, fsys fs.FS, name, ctype string) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// --- WhatsApp endpoints ------------------------------------------------

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	voiceProvider := ""
	if s.cfg.HasSarvam() {
		voiceProvider = "sarvam"
	} else if s.cfg.HasElevenLabs() {
		voiceProvider = "elevenlabs"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"logged_in":    s.wa.LoggedIn(),
		"connected":    s.wa.Connected(),
		"phone":        s.wa.PhoneNumber(),
		"pairing":      s.wa.QRPairing(),
		"last_error":   s.wa.LastError(),
		"owner":        s.cfg.Owner,
		"station":      s.cfg.Station,
		"voice_replies": s.cfg.VoiceReplies,
		"voice_ready":  s.wa.VoiceReplyEnabled(),
		"voice_provider": voiceProvider,
		"stt_ready":    s.wa.STTEnabled(),
		"spotify":      s.spot.Enabled(),
		"gmail":        s.gmail.Enabled(),
	})
}

func (s *Server) qrPNG(w http.ResponseWriter, r *http.Request) {
	png, err := s.wa.QRPNG()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if png == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "no QR available"})
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

func (s *Server) command(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "use POST"})
		return
	}
	var body struct {
		Message string `json:"message"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if strings.TrimSpace(body.Message) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "message required"})
		return
	}
	reply := s.agent.Run(body.Message)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "reply": reply.Text, "tool": reply.Tool})
}

// --- dashboard & radio -------------------------------------------------

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.agent.Dashboard())
}

func (s *Server) radioStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"station":  s.cfg.Station,
		"on_air":   false,
		"listeners": 0,
		"queue_len": len(s.store.Queue()),
	})
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	tracks, err := s.spot.SearchTracks(q, 6)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"results": []any{}, "error": err.Error()})
		return
	}
	out := make([]map[string]any, 0, len(tracks))
	for _, t := range tracks {
		out = append(out, t.ToJSON())
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": out})
}

func (s *Server) queue(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"queue": s.store.Queue()})
	case http.MethodPost:
		var body struct {
			Query string `json:"query"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		reply := s.agent.Run("play " + body.Query)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "reply": reply.Text})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) queueNext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "use POST"})
		return
	}
	q := s.store.Queue()
	var next *store.QueueItem
	for i := range q {
		if q[i].Status == "queued" {
			q[i].Status = "claimed"
			next = &q[i]
			break
		}
	}
	_ = s.store.SaveQueue(q)
	if next == nil {
		writeJSON(w, http.StatusOK, map[string]any{"item": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": next})
}

func (s *Server) queueRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "use POST"})
		return
	}
	var body struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	q := s.store.Queue()
	removed := false
	for i := range q {
		if q[i].ID == body.ID && q[i].Status != "done" {
			q[i].Status = "done"
			removed = true
			break
		}
	}
	_ = s.store.SaveQueue(q)
	writeJSON(w, http.StatusOK, map[string]any{"ok": removed})
}

// --- email --------------------------------------------------------------

func (s *Server) emailStatus(w http.ResponseWriter, r *http.Request) {
	if !s.gmail.Enabled() {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false, "connected": false, "setup_required": true})
		return
	}
	conn := s.gmail.Connection()
	writeJSON(w, http.StatusOK, map[string]any{
		"configured": true,
		"connected":  conn != nil && conn.ConnectionID != "",
		"setup_required": false,
	})
}

func (s *Server) emailConnect(w http.ResponseWriter, r *http.Request) {
	if !s.gmail.Enabled() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "Gmail integration is not configured"})
		return
	}
	link, err := s.gmail.ConnectLink()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	http.Redirect(w, r, link, http.StatusFound)
}

func (s *Server) emailInbox(w http.ResponseWriter, r *http.Request) {
	if !s.gmail.Enabled() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "Gmail integration is not configured"})
		return
	}
	query := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	messages, err := s.gmail.Messages(query, limit)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "messages": messages})
}

// --- music state/command file plumbing -------------------------------

func (s *Server) musicState(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data := map[string]any{}
		_ = s.store.ReadJSONFile("music-state.json", &data)
		writeJSON(w, http.StatusOK, data)
	case http.MethodPost:
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		body["updated_at"] = time.Now().UTC().Format(time.RFC3339)
		s.store.WriteJSONFile("music-state.json", body)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) musicCommand(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data := map[string]any{}
		_ = s.store.ReadJSONFile("music-command.json", &data)
		writeJSON(w, http.StatusOK, map[string]any{"command": data})
	case http.MethodPost:
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		body["id"] = store.NewID("music")
		body["created_at"] = time.Now().UTC().Format(time.RFC3339)
		s.store.WriteJSONFile("music-command.json", body)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "command_id": body["id"]})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) memoryEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "use POST"})
		return
	}
	var body struct {
		Type  string `json:"type"`
		Track string `json:"track"`
		Text  string `json:"text"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch body.Type {
	case "play", "skip", "replay", "conversation", "mood", "preference", "taste":
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unsupported memory event"})
		return
	}
	s.store.RecordMemory(body.Type, body.Track, body.Text)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}