# bday coordinator

A single-page AI birthday coordinator. Sheetal opens it, an agent briefs her on a
project she did not request, and the page itself is the ask. No slides.

```bash
npm start          # → http://localhost:3000
```

No dependencies, no build step. Node 18+.

## How it works

The briefing is a **script**, not a prompt. Every beat is deterministic and lands on
purpose — the joke is in the pacing, so it isn't left to a model. `public/app.js`
holds the script and the client state machine.

She can talk back at any time after the briefing. That part goes to `/api/chat`,
which proxies to any OpenAI-compatible model so the key stays on the server. If no
key is configured, the page degrades to a small pool of canned agent-voice replies
and the scripted briefing is unaffected.

Conversation state lives in `localStorage`, so a refresh resumes where she was.

## Editing it

Everything you'd want to change is in [`shared/config.mjs`](shared/config.mjs):
names, dates, the apartment shortlist, the drinks/food/dessert line, and the
contact button. It is the single source of truth — the browser gets a generated
copy at `public/config.generated.js` (written on every server start, gitignored),
and the server's system prompt reads the same object. Change it once.

Copy text that isn't config — the briefing beats, the good lines, the redacted
strips — lives in `prefix()` and `branch()` in `public/app.js`.

## Env

See [`.env.example`](.env.example). Minimum for chat:

```
LLM_API_KEY=...
LLM_MODEL=gpt-4o-mini
```

Any OpenAI-compatible endpoint works — Groq, OpenRouter, Sarvam, Together, Ollama.
Point `LLM_BASE_URL` and `LLM_CHAT_PATH` at it.

When she taps Friday or Saturday, the last step is a WhatsApp deep link to
`NOTIFY_WHATSAPP` with her chosen day pre-filled, so she just hits send. That
number is injected from the environment at boot, not committed — this repo is
public.

The server reads `PORT` from the environment, which is what Coolify injects, and
binds every interface — deliberately ignoring `HOST`, because Coolify's healthcheck
probes the container over loopback and a platform-injected `HOST` value is not an
address the process can bind. It exposes `GET /healthz` for probes.

## Deploy

Repo: `github.com/vishalgojha/sd`. Dockerfile included, no `npm install` needed.

The Coolify app (`sdsheetal`) builds this Dockerfile, listens on the injected
`PORT`, and probes `/healthz` — all of which work unchanged. The old WhatsApp
environment variables and the `/data` volume are simply ignored now.

Old Go project history: local full clone at `~/bday-backup`.