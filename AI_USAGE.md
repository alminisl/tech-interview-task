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

### Sync layer (automated — passing)

- Wrote a Go integration test (`server/sync_test.go`) that drives the hub through the
  exact behaviours the task lists: client A connects (init, no peers) → client B
  connects and sees A in its init peer list → A sends a camera update and B receives it
  → A disconnects and B receives a `leave` for A. `go test ./...` passes.
- Smoke-tested the running binary with curl: viewer page, `sync.js`, `potree.js` and
  `three.module.js` all return `200`; `/ws` returns `101 Switching Protocols` with a
  valid `Sec-WebSocket-Accept` and immediately pushes the `init` frame
  (`{"type":"init","id":"peer-1","color":"#e6194b","peers":[]}`).
- Verified the frontend's Potree API calls against the submodule source before relying
  on them: `viewer.scene.getActiveCamera()` (scene.js:339), `viewer.scene.scene` is the
  THREE scene Potree renders (scene.js:17), `pointcloud.boundingBox` is a `Box3`
  (PointCloudOctree.js:109).

### Visual two-tab test (needs a browser — human verification)

The one thing that can't be checked headlessly is the WebGL rendering: whether the
cloud is readable with elevation coloring, and whether moving the camera in tab A
visibly moves the cone in tab B (and the cone disappears when A closes). Steps for this
are in the README "How to test with two browser tabs" section.
