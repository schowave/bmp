# Bundesliga Manager Professional

[![Release](https://github.com/schowave/bmp/actions/workflows/release.yml/badge.svg)](https://github.com/schowave/bmp/actions/workflows/release.yml)
[![GitHub Release](https://img.shields.io/github/v/release/schowave/bmp)](https://github.com/schowave/bmp/releases/latest)
[![Docker Image](https://img.shields.io/docker/v/schowave/bmp?sort=semver&label=Docker%20Hub)](https://hub.docker.com/r/schowave/bmp)

The classic 90s DOS football management game — containerized and playable in the browser via [noVNC](https://novnc.com).

<p align="center">
  <img src="images/bmp.png" alt="Bundesliga Manager Professional" width="700">
</p>

## Features

- **Browser-based** — play directly in any browser, no client installation needed
- **Persistent savegames** — game saves are stored on the host via Docker volume (`D:` drive in-game)
- **Auto-updates** — [Watchtower](https://containrrr.dev/watchtower/)-compatible via container labels
- **Optimized for streaming** — tuned DOSBox config for low-latency VNC (640x480, 16-bit, frameskip)

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
make run
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
2. Add the `docker-compose.yml` from this repository
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

Releases are managed via GitHub Actions:

1. Go to **Actions** → **Release** → **Run workflow**
2. Either enter a version number (e.g. `4.1.0`) or leave empty to auto-increment the patch version (e.g. `4.0.1` → `4.0.2`)
3. The workflow updates `VERSION`, creates a git tag, builds the Docker image for `linux/amd64`, and pushes to Docker Hub
4. Watchtower picks up the new image automatically on connected hosts

> Requires GitHub Secrets: `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`

## Development

| Command | Description |
|---|---|
| `make build` | Build the container image |
| `make run` | Stop, build, and start in detached mode |
| `make stop` | Stop and remove the container |
| `make push` | Build and push the `linux/amd64` image to Docker Hub |

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

`AUTOSAVE.MAN`, `BMP/AUTOSAVE.MAN` and `BMP/MICHEL1.MAN` ship with the image, so go by the modification date rather than the name to tell your own save apart — the game overwrites `AUTOSAVE.MAN` as you play, which makes a recent date on it yours.

### The mount has to be writable

The game runs unprivileged as user `bmp` (uid 1000), while the mounted directory belongs to whoever created it on the host. If that is somebody else, saving to `D:` simply fails and the game gives no useful hint. Check it directly:

```bash
docker exec bmp sh -c 'touch /savegame/PROBE && echo writable || echo denied; rm -f /savegame/PROBE'
```

`docker-compose.yml` therefore runs the container in group `100`, which on a Synology is `users`, the group owning the shared folders, and whose ACL grants write access:

```yaml
    group_add:
      - "100"
```

Note that on a Synology the POSIX mode can read `drwxrwxrwx` while writing is still refused, because the ACL decides — `chmod` does not help there. On another host, either use the group that owns the directory or `chown -R 1000:1000 ./savegame` instead.
