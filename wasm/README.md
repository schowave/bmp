# BMP WASM — Browser Edition

Runs Bundesliga Manager Professional in the browser via [js-dos](https://js-dos.com/) (DOSBox in WebAssembly). A small Go server handles static file serving and persists savegames to disk so they survive across browsers and devices.

## Quick Start

```bash
make build     # build the .jsdos bundle
make docker    # build the container image
make run-docker  # run locally with save persistence
```

Then open http://localhost:8090.

## Makefile Targets

| Target        | Description                                      |
|---------------|--------------------------------------------------|
| `build`       | Build `bmp.jsdos` bundle and copy assets         |
| `docker`      | Build the container image (`bmp-wasm`)           |
| `run-docker`  | Run the container locally (port 8090, saves in `/tmp/bmp-saves`) |
| `push`        | Build and push image to Docker Hub               |
| `run`         | Local dev server (python, no save persistence)   |
| `clean`       | Remove generated files                           |

## Architecture

```
Browser (js-dos)
  │
  │  fsChanges.push(data)  ──→  PUT /api/saves/bmp  ──→  /data/bmp.sav
  │  fsChanges.pull(key)   ──→  GET /api/saves/bmp   ──→  /data/bmp.sav
  │
  └──  static files        ──→  GET /  (from /public/)
```

The Go server (`server.go`) is ~65 lines, stdlib-only, zero dependencies:
- Serves static files (index.html, bmp.jsdos, etc.) from `/public/`
- `GET /api/saves/:name` — returns save data (404 if missing)
- `PUT /api/saves/:name` — writes save data to `/data/:name.sav` (max 4 MB)

## Synology NAS Deployment

### Docker Compose (recommended)

```bash
# Copy docker-compose.yml to your NAS, then:
docker compose up -d
```

Save files appear as plain files in the mounted directory, visible in File Station.

### Synology Container Manager Settings

If you prefer the Synology GUI over `docker compose`:

1. **Registry** — search `schowave/bmp`, download the `wasm` tag
2. **Container → Create**
   - Image: `schowave/bmp:wasm`
   - Container name: `bmp`
   - Auto-restart: enabled
3. **Port Settings**
   - Local port: `8090` → Container port: `8080` (TCP)
4. **Volume Settings**
   - Host path: `/volume1/docker/bmp-saves` → Mount path: `/data` (read/write)
5. **Done** — access at `http://<nas-ip>:8090`

### docker run

```bash
docker run -d --restart=unless-stopped \
  --name bmp \
  -p 8090:8080 \
  -v /volume1/docker/bmp-saves:/data \
  schowave/bmp:wasm
```

## Save Files

Saves are stored as `/data/bmp.sav`. They are written whenever js-dos triggers an auto-save (on exit fullscreen, tab blur, or periodic interval).

To back up or restore, simply copy the `.sav` file.
