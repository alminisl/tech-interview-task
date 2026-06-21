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

_(to be filled in as we build)_

---

## Testing phase

_(to be filled in as we test — including the two-tab walkthrough)_
