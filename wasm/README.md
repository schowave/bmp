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
| `run-docker`  | Run the container locally (port 8090, saves in `wasm/savegame`)  |
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
   - Container name: `bmp-wasm`
   - Auto-restart: enabled
3. **Port Settings**
   - Local port: `8090` → Container port: `8080` (TCP)
4. **Volume Settings**
   - Host path: `/volume1/docker/bmp-saves` → Mount path: `/data` (read/write)
5. **Done** — access at `http://<nas-ip>:8090`

### docker run

```bash
docker run -d --restart=unless-stopped \
  --name bmp-wasm \
  -p 8090:8080 \
  -v /volume1/docker/bmp-saves:/data \
  schowave/bmp:wasm
```

## Save Files

Each visitor gets their own save. On the first visit the page draws a slot name from four football terms, such as `konter-flanke-pokal-derby`, keeps it in `localStorage`, and stores the save as `/data/<slot>.sav` on the host, so it survives a browser restart.

Names are meant to be read out, typed and recognised, which is what makes `?slot=` useful. Only nouns are used, so no German adjective ending can come out wrong. Four out of fifty without repetition is about 5.5 million combinations, so no trailing number is needed — but it is far short of a UUID, and someone working through the API could find another player's save.

A drawn name is checked against the server before it is kept, and redrawn up to five times if it is taken. That covers the case that hurts, since the periodic save makes a slot in use visible within a minute. Two gaps remain: a name handed out but never saved is invisible, and checking and claiming are two steps, so two visitors in the same millisecond could draw the same name.

The slot is written into the address bar, so the URL can be bookmarked or shared. Opening it restores that slot's files automatically — there is nothing to import. Sharing it means playing on the same save, not watching: whoever opens the link continues there and overwrites it on the next save.

`?slot=<name>` picks a slot and remembers it. Use it to carry a save to another browser or device, to recover one after clearing site data, or to keep playing an older save — `?slot=bmp` reaches the shared save from before slots existed. The id is shown next to the version at the bottom right; one click selects it. A slot id is not a login: anyone who knows it can read and overwrite that save, which matters little for the random ids and a lot for a name someone could guess.

Old slots are never cleaned up, so the directory grows by one small file per visitor.

Quitting the game inside BMP restarts it instead of dropping to the DOS prompt, and the save is uploaded right then: `START.BAT` in the bundle loops on the game and echoes a marker on exit, which the page picks up from the DOS console and saves immediately. Since that is the moment BMP has just written the savegame, this is the path that matters.

Beyond that the page saves every 60 seconds while the game runs, and again as soon as it goes into the background, so switching tabs or away from the browser keeps the progress. The interval is not the main path any more but the insurance against a crash: saved in-game, kept playing, backend panics — without it that savegame would be lost although BMP had already written it. Saving on close is not possible: building the archive is asynchronous and the request no longer goes out, which was confirmed by measurement. js-dos itself only saves on leaving fullscreen, on releasing the pointer lock, and on `visibilitychange` — it has no periodic save, and nothing on close, which is why the page adds both.

What gets stored is the emulator's filesystem, not the running game state. A game saved from inside BMP is preserved; an unsaved session is not.

Save to drive `D:` in-game. It is mounted from a `SAVES` directory inside the bundle and, like `C:`, is covered by the slot, so either drive would persist here — but the VNC variant only persists `D:`, so using `D:` in both means one instruction rather than two. Saves made on `C:` before this are still there and still work.

To back up or restore, simply copy the `.sav` file.
