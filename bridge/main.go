// Sheetal Bridge executes approved everyday actions on a local Linux or
// Windows machine. It polls the server over outbound HTTPS; no inbound port
// is required. Build with: go build -o sheetal-bridge ./bridge
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type job struct {
	ID         string         `json:"id"`
	Action     string         `json:"action"`
	Parameters map[string]any `json:"parameters"`
}

type settings struct {
	Server string `json:"server"`
	Token  string `json:"token"`
	Device string `json:"device"`
}

func configFile() string {
	if runtime.GOOS == "windows" {
		if d := os.Getenv("APPDATA"); d != "" {
			return filepath.Join(d, "SheetalBridge", "config.json")
		}
	}
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "sheetal-bridge", "config.json")
	}
	return ""
}

func readSettings() settings {
	var s settings
	if p := configFile(); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			_ = json.Unmarshal(b, &s)
		}
	}
	return s
}

func main() {
	server := flag.String("server", os.Getenv("SHEETAL_SERVER"), "Sheetal URL")
	token := flag.String("token", os.Getenv("SHEETAL_BRIDGE_TOKEN"), "bridge token")
	device := flag.String("device", "laptop", "device id")
	flag.Parse()
	stored := readSettings()
	if *server == "" {
		*server = stored.Server
	}
	if *token == "" {
		*token = stored.Token
	}
	if *device == "laptop" && stored.Device != "" {
		*device = stored.Device
	}
	if *server == "" || *token == "" {
		fmt.Fprintln(os.Stderr, "set SHEETAL_SERVER and SHEETAL_BRIDGE_TOKEN")
		os.Exit(2)
	}
	client := &http.Client{Timeout: 35 * time.Second}
	for {
		var out struct {
			Job *job `json:"job"`
		}
		req, _ := http.NewRequest("POST", strings.TrimRight(*server, "/")+"/api/bridge/next?device_id="+url.QueryEscape(*device), nil)
		req.Header.Set("Authorization", "Bearer "+*token)
		resp, err := client.Do(req)
		if err == nil {
			_ = json.NewDecoder(resp.Body).Decode(&out)
			resp.Body.Close()
		}
		if out.Job != nil {
			status, result := run(out.Job.Action, out.Job.Parameters)
			body, _ := json.Marshal(map[string]any{"id": out.Job.ID, "status": status, "result": result})
			r, _ := http.NewRequest("POST", strings.TrimRight(*server, "/")+"/api/bridge/result", bytes.NewReader(body))
			r.Header.Set("Authorization", "Bearer "+*token)
			r.Header.Set("Content-Type", "application/json")
			rr, _ := client.Do(r)
			if rr != nil {
				rr.Body.Close()
			}
		}
		time.Sleep(2 * time.Second)
	}
}
func run(action string, p map[string]any) (string, string) {
	s := func(k string) string { v, _ := p[k].(string); return v }
	switch strings.ToLower(action) {
	case "notify":
		title, message := s("title"), s("message")
		if title == "" { title = "Agent V" }
		if message == "" { return "failed", "notification message required" }
		if runtime.GOOS == "windows" {
			_ = exec.Command("msg", "*", title+": "+message).Run()
			return "done", "notification shown"
		}
		if err := exec.Command("notify-send", title, message).Run(); err != nil { return "failed", err.Error() }
		return "done", "notification shown"
	case "open_url":
		u := s("url")
		if u == "" {
			return "failed", "url required"
		}
		if runtime.GOOS == "windows" {
			_ = exec.Command("cmd", "/c", "start", "", u).Start()
		} else {
			_ = exec.Command("xdg-open", u).Start()
		}
		return "done", "opened " + u
	case "open_app":
		n := s("name")
		if n == "" {
			n = s("app")
		}
		if runtime.GOOS == "windows" {
			_ = exec.Command("cmd", "/c", "start", "", n).Start()
		} else {
			_ = exec.Command(n).Start()
		}
		return "done", "launched " + n
	case "type_text":
		t := s("text")
		if runtime.GOOS == "linux" {
			_ = exec.Command("xdotool", "type", "--clearmodifiers", t).Run()
			return "done", "typed text"
		}
		return "failed", "Windows text input requires the companion UI automation module"
	case "press_key":
		k := s("key")
		if runtime.GOOS == "linux" {
			_ = exec.Command("xdotool", "key", k).Run()
			return "done", "pressed " + k
		}
		return "failed", "Windows key input requires the companion UI automation module"
	default:
		return "failed", "unsupported action: " + action
	}
}
