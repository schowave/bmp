# Bundesliga Manager Professional

[![Release](https://github.com/schowave/bmp/actions/workflows/release.yml/badge.svg)](https://github.com/schowave/bmp/actions/workflows/release.yml)
[![GitHub Release](https://img.shields.io/github/v/release/schowave/bmp)](https://github.com/schowave/bmp/releases/latest)
[![Docker Image](https://img.shields.io/docker/v/schowave/bmp?sort=semver&label=Docker%20Hub)](https://hub.docker.com/r/schowave/bmp)

The classic 90s DOS football management game — containerized and playable in the browser via [noVNC](https://novnc.com).

<p align="center">
  <img src="docs/bmp.png" alt="Bundesliga Manager Professional" width="700">
</p>

## Features

- **Browser-based** — play directly in any browser, no client installation needed
- **Persistent savegames** — game saves are stored on the host via Docker volume (`D:` drive in-game)
- **Auto-updates** — [Watchtower](https://containrrr.dev/watchtower/)-compatible via container labels
- **Optimized for streaming** — tuned DOSBox config for low-latency VNC (640x480, 16-bit, frameskip)
- **Idle when unused** — DOSBox is paused while no browser is connected, so the container uses next to no CPU between sessions

## Quick Start

### Docker

```bash
docker run -d \
  -v ./savegame:/savegame \
  -p 8080:8080 \
  schowave/bmp:latest
```

Open [http://localhost:8080](http://localhost:8080)

### From Source

```bash
git clone https://github.com/schowave/bmp.git
cd bmp
mise run vnc:run
```

## Architecture

```
Browser (noVNC) ──WebSocket──▸ websockify :8080 ──▸ TigerVNC :5901
                                                       │
                                                  Ratpoison WM
                                                       │
                                                  DOSBox 0.74-3
                                                   ├── C: /dos/bmp     (game files)
                                                   └── D: /savegame    (persistent saves)
```

## Deployment

### Synology NAS

1. Create a project folder on your NAS (e.g. `/volume1/docker/bmp/`)
2. Add `vnc/docker-compose.yml` from this repository
3. In **Container Manager** → **Project** → **Create**, point to the folder and start
4. If using a reverse proxy, add WebSocket headers under **Custom Header**:

   | Header | Value |
   |---|---|
   | `Upgrade` | `$http_upgrade` |
   | `Connection` | `$connection_upgrade` |

The container is labeled for [Watchtower](https://containrrr.dev/watchtower/) — if a Watchtower instance is running on the NAS, it will automatically pull new images on release.

### Other Platforms

The Docker image `schowave/bmp` is built for `linux/amd64`.

```yaml
services:
  bmp:
    image: schowave/bmp:latest
    ports:
      - "8080:8080"
    volumes:
      - ./savegame:/savegame
    # The group that owns ./savegame on the host, so the game may write there.
    # See Savegames below.
    group_add:
      - "100"
    restart: unless-stopped
```

## Releases

Releases are managed via GitHub Actions, and only there — there is no local push:

1. Go to **Actions** → **Release** → **Run workflow**, or run `mise run release [version]` (needs the `gh` CLI)
2. Either enter a version number (e.g. `4.1.0`) or leave empty to auto-increment the patch version (e.g. `4.0.1` → `4.0.2`). The version must be above the latest tag and not taken yet.
3. The workflow runs CI first, then builds both images for `linux/amd64` and pushes them under the version tag alone
4. Only once both images are on Docker Hub does it commit `VERSION`, push the git tag, and move `latest` and `wasm` to the new images
5. Watchtower picks up the new image automatically on connected hosts

A run that fails or is cancelled before step 4 leaves no git tag behind and does not change what Watchtower pulls. Only one release runs at a time.

> Requires GitHub Secrets: `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`

### CI

Every push to `main` and every pull request builds both images and runs the VNC benchmark with `--check` against the VNC image: the picture has to fill the screen, clicks have to arrive, and the container must go idle without a client. The screenshot is attached to the run. See [vnc/bench/README.md](vnc/bench/README.md).

The WASM image gets a smoke test without a browser, `wasm/smoke.sh`: page and bundle are served, a save comes back byte for byte, invalid names and oversized saves are refused, and the server does not run as root. Locally: `mise run wasm:smoke`.

## Development

Tasks are defined in `mise.toml` and run with [mise](https://mise.jdx.dev/). `mise tasks` lists them all, including the `wasm:*` tasks for the WASM image.

### Layout

| Path | Contents |
|---|---|
| `bmp/` | Game files, used by both images |
| `web/` | Help page and favicon, served by both images |
| `vnc/` | VNC image: `Dockerfile`, start script, DOSBox config, noVNC player page, `docker-compose.yml` |
| `vnc/bench/` | Benchmark for the VNC image with Playwright, see [vnc/bench/README.md](vnc/bench/README.md) |
| `wasm/` | WASM image, see [wasm/README.md](wasm/README.md) |
| `docs/` | Images for this README |

Both Dockerfiles build with the repository root as context, e.g. `docker build -f vnc/Dockerfile .`

| Command | Description |
|---|---|
| `mise run vnc:build` | Build the container image |
| `mise run vnc:run` | Stop, build, and start in detached mode |
| `mise run vnc:stop` | Stop and remove the container |
| `mise run vnc:bench [label]` | Build and benchmark the image with checks, see [vnc/bench/README.md](vnc/bench/README.md) |
| `mise run release` | Start the release workflow, see [Releases](#releases) |

## Help page

Both images ship a German quick reference for the game at `/hilfe.html`, linked from the
bottom right of the game page. The WASM variant also answers the shorter `/hilfe`; the VNC
one does not, because websockify serves files literally, which is why both pages link to
`hilfe.html`.

The page lives at `web/hilfe.html` since both images serve it; both Docker builds take it
from there.

## Savegames

The game mounts two DOS drives:

| Drive | Mount | Purpose |
|---|---|---|
| `C:` | `/dos/bmp` | Game files (inside container) |
| `D:` | `/savegame` | Persistent saves (host directory) |

Save to `D:` in-game. A save placed on `C:` lives inside the container and is gone with the next image, so it does not survive an update. To rescue one from a running container before rebuilding it:

```bash
docker exec bmp find /dos -iname '*.MAN' -exec ls -la {} \;
docker cp bmp:/dos/bmp/YOURSAVE.MAN ./savegame/YOURSAVE.MAN
```

`AUTOSAVE.MAN` ships with the image, so go by the modification date rather than the name to tell your own save apart — the game overwrites `AUTOSAVE.MAN` as you play, which makes a recent date on it yours.

### The mount has to be writable

The game runs unprivileged as user `bmp` (uid 1000), while the mounted directory belongs to whoever created it on the host. If that is somebody else, saving to `D:` simply fails and the game gives no useful hint. Check it directly:

```bash
docker exec bmp sh -c 'touch /savegame/PROBE && echo writable || echo denied; rm -f /savegame/PROBE'
```

`vnc/docker-compose.yml` therefore runs the container in group `100`, which on a Synology is `users`, the group owning the shared folders, and whose ACL grants write access:

```yaml
    group_add:
      - "100"
```

Note that on a Synology the POSIX mode can read `drwxrwxrwx` while writing is still refused, because the ACL decides — `chmod` does not help there. On another host, either use the group that owns the directory or `chown -R 1000:1000 ./savegame` instead.
