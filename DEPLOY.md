# Deploy Sheetal (`sd`) to https://sd.vishalojha.me

Deployed form: sd.vishalojha.me (coolify.propai.live).

## Option A — One click in Coolify (recommended)

1. Open https://coolify.propai.live → log in
2. App: **sdsheetal** (`kjhp61nafpugk27griupkfqi`, project `im05yd2psmhwyzj8q8tl086b`)
3. Under **Settings → Source**, change repository from `vishalgojha/rjsheetal` →
   `https://github.com/vishalgojha/sd`
4. Keep build method as **Dockerfile** (overrides py buildpack); build command:
   `docker buildx build . -t ghcr.io/vishalgojha/sdsheetal:latest`
5. Ensure **Persistent Storage** path = `/data`
6. Click **Deploy** → wait ~2 min

## Option B — Deploy using the built-in Dockerfile

Copy-paste into the deploy host (or any x64 host with Docker):

```bash
# 1) clone the prepared repo
git clone https://github.com/vishalgojha/sd.git /opt/sdsheetal
cd /opt/sdsheetal
# 2) copy existing secrets / data
cp /data/.env.example /data/.env
#    edit /data/.env with your keys (NANGO_SECRET_KEY, SARVAM_ API_KEY, ELEVENLABS_API_KEY, etc.)
# 3) rebuild
docker buildx build . -t sdsheetal:latest
# 4) start (if replacing running container)
docker run -d \
  --name sdsheetal \
  --restart unless-stopped \
  -v /data:/data \
  -p 8080:8080 \
  sdsheetal:latest
```

## Verification

- `https://sd.vishalojha.me/` → returns 200 **without** the `RJSheetal/1.0 Python` header
- `https://sd.vishalojha.me/healthz` →`{"ok": true}`
- `https://sd.vishalojha.me/api/whatsapp/status` → includes `{"voice_provider":"sarvam", ...}`
- `https://sd.vishalojha.me/` smoke: confirm the tabbed UI (Status / Chat / Tasks / Music / Memory / Email)
