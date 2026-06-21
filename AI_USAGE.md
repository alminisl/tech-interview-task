# AI Usage Log

A running log of how AI (Claude) was used on this project, kept phase by phase as
the work happened. This is the raw material for the README "Approach" section and a
record for the interview walkthrough.

Stack decided up front: **Potree** frontend (served statically over HTTP),
**Go** WebSocket hub backend, browser tabs as collaborating clients.

---

## Planning phase

**What I asked AI to do**
- Review the task and call out the traps before any code: the 2 GB SoFi file, COPC
  HTTP range loading, elevation-coloring-on-load, peer cleanup on disconnect, and the
  coordinate-frame problem (peer camera state only makes sense if both clients share
  the same origin/offset).
- Confirm the current Potree COPC loading API instead of guessing — verified against
  Potree's own `examples/copc.html`: `Potree.loadPointCloud(url, name, cb)` loads
  `.copc.laz` directly via range requests, and `material.activeAttributeName = "elevation"`
  sets the coloring in the load callback.
- Decide the backend: chose Go (gorilla/websocket hub) over Node for a cleaner
  broadcast model and to show range.

**What I verified manually**
- (pending) Potree builds and serves on this machine (Windows, Node 22).
- (pending) SoFi COPC actually loads and renders readably with elevation coloring.

**Decisions made**
- Build order: get the viewer loading + readable FIRST, then add sync. No camera-sync
  code until a single tab renders the cloud correctly.
- Commit in stages to show evolution.

---

## Implementation phase

### Step 1 — viewer loads and is readable

**What I asked AI to do**
- Pull Potree in as a git submodule (keeps our repo to just our own code, pins a
  Potree version) and build it. Verified `libs/copc` ships in the build, so COPC is
  supported without conversion.
- Write our own viewer page (`frontend/index.html`) from Potree's `examples/copc.html`,
  changing two things: elevation coloring is set unconditionally on load (requirement),
  and the source defaults to the SoFi URL with a `?src=local` override for fast
  iteration against the small bundled `lion_takanawa.copc.laz`.
- Write a single Go server (`server/main.go`) that serves the project root. Chosen
  because `http.FileServer` answers HTTP Range requests (what COPC needs) and the same
  server will later host the `/ws` WebSocket — one origin, no CORS, one command to run.

**What I verified manually / with tooling**
- Go server builds and runs.
- `curl` against the running server: viewer page `200`, `potree.js` `200`, and a
  `Range: bytes=0-99` request on the local COPC sample returns `206 Partial Content`
  with `Content-Range: bytes 0-99/2735983` — range streaming confirmed.
- (pending, needs a browser) SoFi renders readably with elevation coloring.

---

## Testing phase

_(to be filled in as we test — including the two-tab walkthrough)_
