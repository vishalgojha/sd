// Package whatsapp wires the whatsmeow WhatsApp Web client into the assistant:
// QR-code pairing manager, connection lifecycle, and the incoming-message
// handler that runs the agent and sends text/voice replies.
package whatsapp

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	wastore "go.mau.fi/whatsmeow/store"

	"github.com/vishalgojha/sdsheetal/internal/assistant"
	"github.com/vishalgojha/sdsheetal/internal/config"
	"github.com/vishalgojha/sdsheetal/internal/sarvam"
	"github.com/vishalgojha/sdsheetal/internal/store"
	"github.com/vishalgojha/sdsheetal/internal/tts"
)

// Client wraps the whatsmeow client with the assistant glue.
type Client struct {
	cfg     *config.Config
	store   *store.Store
	agent   *assistant.Agent
	tts     *tts.Client
	sarvam  *sarvam.Client
	wac     *whatsmeow.Client
	device  *wastore.Device
	mu      sync.RWMutex
	qrCode  string // latest pairing QR (plain text to encode)
	qrTime  time.Time
	pairing bool
	lastErr string
	sentMu  sync.Mutex
	sentIDs map[types.MessageID]struct{}
}

// waLogger implements whatsmeow's waLog.Logger over the standard logger.
type waLogger struct{}

func (waLogger) Debugf(format string, args ...any) { log.Printf("WA: "+format, args...) }
func (waLogger) Infof(format string, args ...any)  { log.Printf("WA: "+format, args...) }
func (waLogger) Warnf(format string, args ...any)  { log.Printf("WA: "+format, args...) }
func (waLogger) Errorf(format string, args ...any) { log.Printf("WA: "+format, args...) }
func (waLogger) Sub(string) waLog.Logger           { return waLog.Noop }

func New(cfg *config.Config, st *store.Store, agent *assistant.Agent, voice *tts.Client, sv *sarvam.Client) *Client {
	return &Client{cfg: cfg, store: st, agent: agent, tts: voice, sarvam: sv, sentIDs: make(map[types.MessageID]struct{})}
}

// StorePath is the sqlite file used for the whatsmeow session.
func (c *Client) StorePath() string { return c.cfg.WhatsAppStore }

func (c *Client) raw() *whatsmeow.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.wac
}

// LoggedIn reports whether a device is already stored.
func (c *Client) LoggedIn() bool {
	cc := c.raw()
	return cc != nil && cc.Store.ID != nil
}

// Connected reports the live socket state.
func (c *Client) Connected() bool {
	cc := c.raw()
	return cc != nil && cc.IsConnected()
}

// PhoneNumber returns the logged-in number (digits only) if known.
func (c *Client) PhoneNumber() string {
	cc := c.raw()
	if cc != nil && cc.Store.ID != nil {
		return cc.Store.ID.User
	}
	return ""
}

// LastError returns the most recent connection error text.
func (c *Client) LastError() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastErr
}

// QRPairing reports whether pairing mode is active.
func (c *Client) QRPairing() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.pairing
}

// QRPNG renders the current pairing QR as a PNG image (or nil).
func (c *Client) QRPNG() ([]byte, error) {
	c.mu.RLock()
	code := c.qrCode
	recent := time.Since(c.qrTime) < 30*time.Second
	c.mu.RUnlock()
	if code == "" || !recent {
		return nil, nil
	}
	return qrcode.Encode(code, qrcode.Medium, 300)
}

// Start connects a stored session if present, otherwise enters pairing mode.
func (c *Client) Start() error {
	container, err := sqlstore.New(context.Background(), "sqlite3", "file:"+c.StorePath()+"?_foreign_keys=on", waLogger{})
	if err != nil {
		return fmt.Errorf("failed to open whatsmeow store: %w", err)
	}
	device, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to read whatsmeow device: %w", err)
	}
	c.mu.Lock()
	c.device = device
	c.mu.Unlock()

	client := whatsmeow.NewClient(device, waLogger{})
	client.AddEventHandler(c.handleEvent)
	c.mu.Lock()
	c.wac = client
	c.mu.Unlock()

	if device.ID != nil {
		log.Printf("whatsmeow: stored session found for %s", device.ID.User)
		if err := client.Connect(); err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	} else {
		log.Printf("whatsmeow: no stored session — starting pairing mode")
		go c.startPairing()
	}
	return nil
}

// Disconnect shuts the client down.
func (c *Client) Disconnect() {
	c.mu.Lock()
	c.pairing = false
	c.qrCode = ""
	c.mu.Unlock()
	if cc := c.raw(); cc != nil {
		cc.Disconnect()
	}
}

func (c *Client) startPairing() {
	client := c.raw()
	if client == nil {
		return
	}
	qr, err := client.GetQRChannel(context.Background())
	if err != nil {
		c.mu.Lock()
		c.pairing = false
		c.lastErr = err.Error()
		c.mu.Unlock()
		log.Printf("whatsmeow: QR channel failed: %v", err)
		return
	}
	c.mu.Lock()
	c.pairing = true
	c.lastErr = ""
	c.mu.Unlock()

	if err := client.Connect(); err != nil {
		c.mu.Lock()
		c.pairing = false
		c.lastErr = err.Error()
		c.mu.Unlock()
		log.Printf("whatsmeow: connect failed during pairing: %v", err)
		return
	}

	for item := range qr {
		switch item.Event {
		case whatsmeow.QRChannelEventCode:
			c.mu.Lock()
			c.qrCode = item.Code
			c.qrTime = time.Now()
			c.mu.Unlock()
		case whatsmeow.QRChannelEventError:
			c.mu.Lock()
			c.lastErr = item.Error.Error()
			c.mu.Unlock()
			log.Printf("whatsmeow: pairing error: %v", item.Error)
		default:
			c.mu.Lock()
			c.pairing = false
			c.qrCode = ""
			c.mu.Unlock()
		}
	}
}

// handleEvent is whatsmeow's incoming event pump.
func (c *Client) handleEvent(raw any) {
	switch evt := raw.(type) {
	case *events.Message:
		c.onMessage(evt)
	case *events.Connected:
		c.mu.Lock()
		c.pairing = false
		c.lastErr = ""
		c.mu.Unlock()
		log.Printf("whatsmeow: connected as %s", c.PhoneNumber())
	case *events.Disconnected:
		log.Printf("whatsmeow: disconnected (auto-reconnect active)")
	case *events.LoggedOut:
		log.Printf("whatsmeow: logged out; will require re-pairing")
	case *events.QRScannedWithoutMultidevice:
		log.Printf("whatsmeow: QR scanned without multi-device — ignoring")
	}
}

// onMessage handles a single incoming message from a direct chat.
func (c *Client) onMessage(evt *events.Message) {
	// WhatsApp marks messages sent to your own "Message yourself" chat as
	// IsFromMe too. Allow those through, but ignore messages the assistant
	// itself just sent so self-chat cannot create a reply loop.
	if evt.Info.IsFromMe && !c.isSelfChat(evt.Info.Chat) {
		return
	}
	if evt.Info.IsFromMe && c.wasSent(evt.Info.ID) {
		return
	}
	jid := evt.Info.Chat
	if jid.Server != types.DefaultUserServer {
		return // only DMs; groups ignored
	}
	if !c.allowed(jid) {
		return
	}
	if evt.Message == nil {
		return
	}

	// Voice notes / audio: transcribe with Sarvam STT, then treat as text.
	if text := extractText(evt); text == "" {
		if audio := evt.Message.GetAudioMessage(); audio != nil && c.sarvam != nil && c.sarvam.Enabled() {
			go c.handleVoiceNote(evt, jid)
		}
		return
	}
	text := extractText(evt)
	log.Printf("whatsmeow: message from %s: %q", jid.User, text)

	client := c.raw()
	reply := c.agent.Run(text)
	if reply.Text == "" {
		return
	}

	resp, err := client.SendMessage(context.Background(), jid, textMessage(reply.Text))
	if err == nil {
		c.rememberSent(resp.ID)
	}
	if c.cfg.VoiceReplies || c.cfg.ReplyVoiceNotes {
		go c.sendVoiceNote(jid, reply.Text)
	}
}

// handleVoiceNote downloads an incoming audio/PTT message, transcribes it via
// Sarvam STT, and runs the transcript through the agent. Runs in a goroutine.
func (c *Client) handleVoiceNote(evt *events.Message, jid types.JID) {
	client := c.raw()
	if client == nil {
		return
	}
	audio := evt.Message.GetAudioMessage()
	data, err := client.Download(context.Background(), audio)
	if err != nil {
		log.Printf("whatsmeow: audio download failed: %v", err)
		return
	}
	mime := audio.GetMimetype()
	text, err := c.sarvam.Transcribe(context.Background(), "voice"+extFromMIME(mime), mime, data)
	if err != nil {
		log.Printf("whatsmeow: transcription failed: %v", err)
		resp, sendErr := client.SendMessage(context.Background(), jid,
			textMessage("I couldn't hear that clearly — could you type it or try again?"))
		if sendErr == nil {
			c.rememberSent(resp.ID)
		}
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	log.Printf("whatsmeow: voice note from %s transcribed: %q", jid.User, text)

	reply := c.agent.Run(text)
	if reply.Text == "" {
		return
	}
	resp, err := client.SendMessage(context.Background(), jid, textMessage(reply.Text))
	if err == nil {
		c.rememberSent(resp.ID)
	}
	if c.cfg.VoiceReplies || c.cfg.ReplyVoiceNotes {
		go c.sendVoiceNote(jid, reply.Text)
	}
}

func extFromMIME(mime string) string {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "audio/ogg", "audio/ogg; codecs=opus", "audio/opus":
		return ".ogg"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/wav", "audio/wave":
		return ".wav"
	case "audio/mp4", "audio/m4a":
		return ".m4a"
	case "audio/amr":
		return ".amr"
	default:
		return ".audio"
	}
}

// allowed checks the configured owner number (if any).
func (c *Client) allowed(jid types.JID) bool {
	owners := strings.FieldsFunc(c.cfg.Owner, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
	})
	if len(owners) == 0 {
		return true
	}
	number := normalizeNumber(jid.User)
	for _, owner := range owners {
		if normalized := normalizeNumber(owner); normalized != "" && normalized == number {
			return true
		}
	}
	return false
}

func (c *Client) isSelfChat(jid types.JID) bool {
	phone := c.PhoneNumber()
	return phone != "" && normalizeNumber(jid.User) == normalizeNumber(phone)
}

func (c *Client) rememberSent(id types.MessageID) {
	if id == "" {
		return
	}
	c.sentMu.Lock()
	c.sentIDs[id] = struct{}{}
	c.sentMu.Unlock()
}

func (c *Client) wasSent(id types.MessageID) bool {
	c.sentMu.Lock()
	defer c.sentMu.Unlock()
	if _, ok := c.sentIDs[id]; !ok {
		return false
	}
	delete(c.sentIDs, id)
	return true
}

func normalizeNumber(s string) string {
	s = strings.TrimPrefix(strings.TrimSpace(s), "+")
	var out strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func extractText(evt *events.Message) string {
	if evt.Message == nil {
		return ""
	}
	if conv := evt.Message.GetConversation(); conv != "" {
		return conv
	}
	if ext := evt.Message.GetExtendedTextMessage(); ext != nil && ext.GetText() != "" {
		return ext.GetText()
	}
	return "" // media ignored in v1
}

// textMessage builds an outgoing text message.
func textMessage(text string) *waE2E.Message {
	return &waE2E.Message{Conversation: &text}
}

// sendVoiceNote synthesizes the reply as speech and sends it as an audio
// message. Sarvam (Indic) is preferred when configured, otherwise ElevenLabs.
// Runs in its own goroutine.
func (c *Client) sendVoiceNote(to types.JID, text string) {
	audio, err := c.synthesize(text)
	if err != nil {
		log.Printf("whatsapp: TTS failed: %v", err)
		return
	}
	client := c.raw()
	if client == nil {
		return
	}
	resp, err := client.Upload(context.Background(), audio, whatsmeow.MediaAudio)
	if err != nil {
		log.Printf("whatsapp: audio upload failed: %v", err)
		return
	}
	secs := uint32(len(audio) / 2048)
	ptt := true
	msg := &waE2E.Message{
		AudioMessage: &waE2E.AudioMessage{
			URL:           &resp.URL,
			Mimetype:      strptr("audio/mpeg"),
			FileLength:    &resp.FileLength,
			Seconds:       &secs,
			PTT:           &ptt,
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			DirectPath:    &resp.DirectPath,
		},
	}
	sendResp, err := client.SendMessage(context.Background(), to, msg)
	if err != nil {
		log.Printf("whatsapp: voice note send failed: %v", err)
	} else {
		c.rememberSent(sendResp.ID)
	}
}

// synthesize picks a TTS provider: Sarvam (Indic voices) if configured,
// otherwise ElevenLabs.
func (c *Client) synthesize(text string) ([]byte, error) {
	if c.sarvam != nil && c.sarvam.Enabled() {
		return c.sarvam.Speak(text)
	}
	if c.tts == nil {
		return nil, fmt.Errorf("no TTS provider configured")
	}
	return c.tts.Speak(text)
}

// VoiceReplyEnabled reports whether a spoken reply should be produced.
func (c *Client) VoiceReplyEnabled() bool {
	if c.sarvam != nil && c.sarvam.Enabled() {
		return true
	}
	return c.tts != nil && c.tts.Enabled()
}

// STTEnabled reports whether incoming voice-note transcription is available.
func (c *Client) STTEnabled() bool {
	return c.sarvam != nil && c.sarvam.Enabled()
}

func strptr(s string) *string { return &s }
