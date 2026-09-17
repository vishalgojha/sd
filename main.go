// SD Sheetal — WhatsApp personal assistant.
//
// A Go rewrite of the sdsheetal Python assistant, carried on WhatsApp via
// whatsmeow. The web UI gives owner pairing (QR scan), a live status panel,
// a command console that drives the same agent, and manual tools.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/vishalgojha/sdsheetal/internal/assistant"
	"github.com/vishalgojha/sdsheetal/internal/config"
	"github.com/vishalgojha/sdsheetal/internal/gmail"
	"github.com/vishalgojha/sdsheetal/internal/sarvam"
	"github.com/vishalgojha/sdsheetal/internal/spotify"
	"github.com/vishalgojha/sdsheetal/internal/store"
	"github.com/vishalgojha/sdsheetal/internal/tts"
	"github.com/vishalgojha/sdsheetal/internal/web"
	"github.com/vishalgojha/sdsheetal/internal/whatsapp"
)

func main() {
	cfg := config.Load()

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("data dir: %v", err)
	}

	st := store.New(cfg.DataDir)
	sp := spotify.New(cfg)
	voice := tts.New(cfg)
	sv := sarvam.New(cfg)
	gm := gmail.New(cfg)
	agent := assistant.New(cfg, st, sp, gm)

	wa := whatsapp.New(cfg, st, agent, voice, sv)
	if err := wa.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "whatsapp start:", err)
		os.Exit(1)
	}

	srv := web.New(cfg, st, agent, sp, gm, wa)

	httpSrv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("sdsheetal (whatsmeow agent) listening on :%s", cfg.Port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Printf("shutting down…")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	wa.Disconnect()
	_ = httpSrv.Shutdown(ctx)
}