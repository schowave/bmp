# VNC benchmark

Measures one configuration of the VNC image end to end: CPU per process inside the
container, WebSocket traffic, and the time from a mouse click in the browser to the
changed picture on the noVNC canvas. The browser is a headless Chromium driven by
Playwright.

```bash
mise run vnc:bench [label]
```

builds the image as `bmp`, runs the benchmark against it with `--check` and prints
all runs so far. For variants, call the script directly:

```bash
cd vnc/bench
npm ci && npx playwright install chromium
node bench.mjs --label cycles10k --conf variants/cycles10k.conf
node bench.mjs --label no-pause --cmd "$(cat variants/no-pause.sh)"
node report.mjs
```

Each run starts its own container `bmp-bench` on port 18080, removes it afterwards
(`--keep` leaves it running), appends one line to `results.jsonl` and saves a
screenshot to `shots/<label>.png`.

| Option | Meaning |
|---|---|
| `--image` | Image to run, default `bmp` (what `mise run vnc:build` builds) |
| `--conf` | DOSBox config mounted over the one in the image |
| `--cmd` | Container command instead of the image's `CMD` |
| `--secs` | Length of each CPU measurement, default 20 |
| `--clicks` | Number of latency probes, default 20 |
| `--check` | Exit with 1 if the run looks broken, see below |
| `--keep` | Leave the container running afterwards |

The container is run with `podman`; set `CONTAINER_CLI=docker` to use Docker.

`variants/` holds the alternatives measured so far: two DOSBox configs with fixed
cycles, and `no-pause.sh`, the container command from before `vnc/start.sh` stopped
DOSBox while nobody is connected.

`shot.mjs` only takes a screenshot, optionally after clicks or key presses, which
helps to find coordinates: `node shot.mjs out.png click:627,135 wait:1000 Enter`.

## What a run does

1. Starts the container and waits 15 s until the game shows the team selection.
2. **No client**: CPU over `--secs` with nobody connected.
3. **Client idle**: opens the page, waits for the first picture, measures CPU and
   WebSocket traffic while nothing happens on screen.
4. **Clicking**: clicks the right arrow of the team strip `--clicks` times and
   measures, for each click, how long until the strip on the canvas changes. This
   covers the whole path: browser, websockify, Xvnc, DOSBox, the game, and back.
5. **Picture size**: the bounding box of the non-black pixels on the canvas. It must
   be the full 640x480; anything smaller means DOSBox no longer scales the game's
   320x200 mode. Also look at the screenshot in `shots/`.
6. **After disconnect**: closes the browser and measures CPU again.

CPU is given in percent of one core, from `utime + stime` in `/proc/<pid>/stat`.

## Checks

With `--check` the run fails if the picture does not fill the 640x480 screen, a
click did not change the picture, DOSBox is not running while a client is
connected, or the container uses more than 5 percent of a core without a client.
CI runs this on every push, see `.github/workflows/ci.yml`. The `no-pause` variant
fails the last check by design.

## Caveats

- On an Apple Silicon Mac the image is built for arm64, while production runs the
  amd64 image. DOSBox's dynamic core differs between the two, so absolute CPU
  numbers do not carry over. Compare runs with each other, not with the production
  host.
- Other load on the host shows up in the numbers. One run in a series came out ten
  points below its repetitions. Run each variant at least twice before drawing a
  conclusion.
- The median click latency jumps between about 31 and 45 ms from run to run,
  whatever the configuration; the 90th percentile stays at 48 ms. Compare p90.
- The click latency includes a few milliseconds of Playwright and CDP overhead. It
  is the same for every run, so differences between runs are meaningful.
