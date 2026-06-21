# Collaborative Point Cloud Viewer

Multiple browser tabs load the same COPC point cloud and share camera position and
view direction in real time. Each tab shows where the other tabs are looking in 3D via
a view cone.


## Stack

- **Frontend:** [Potree](https://github.com/potree/potree) WebGL point cloud viewer,
  loading the SoFi Stadium COPC file directly over HTTP range requests.
- **Backend:** Go WebSocket hub that fans out camera state between connected clients.

## Prerequisites

- [Go](https://go.dev/) 1.21+
- [Node.js](https://nodejs.org/) 18+ and npm (only needed once, to build Potree)
- Git (the Potree viewer is pulled in as a submodule)

## How to run

```bash
# 1. Clone with the Potree submodule
git clone --recurse-submodules <your-repo-url>
cd interview-cyclomedia
# (if you already cloned without --recurse-submodules)
git submodule update --init --recursive

# 2. Build Potree once (downloads its deps and builds build/potree)
cd potree
npm install
cd ..

# 3. Run the server (serves the frontend AND the WebSocket hub)
cd server
go run .
```

Then open <http://localhost:8080/frontend/>.

The one Go server hosts both the static frontend (with HTTP range support, which COPC
streaming needs) and the `/ws` WebSocket hub — so there's a single command and a single
origin, no CORS.

By default the viewer loads the SoFi Stadium COPC file straight from S3
(`https://s3.amazonaws.com/hobu-lidar/sofi.copc.laz`, ~2 GB — expect a slow first load).
For fast local iteration, append `?src=local` to load a tiny sample shipped with Potree:
<http://localhost:8080/frontend/?src=local>.

## How to test with two browser tabs

1. Start the server and open <http://localhost:8080/frontend> in **tab A**
2. Open the same URL in **tab B**.
3. Move/orbit the camera in tab A → within ~1 second a colored view cone in tab B moves
   to show where tab A is looking. Do the same in B → A sees B's cone.
4. Close tab A → its cone disappears from tab B.

To verify the realtime layer on its own (no browser), run the server tests:

```bash
cd server
go test ./...
```

## Approach

### How I scoped it

I built it in the order the requirements depend on each other, and committed in stages
so the history shows the evolution:

1. **Viewer first.** Get the COPC cloud loading and readable (elevation coloring on by
   default) *before* writing any sync code — a broken viewer would make debugging the
   sync layer miserable.
2. **Sync layer.** A WebSocket hub that handles multiple clients, new-joiner state, live
   updates, and disconnect cleanup.
3. **Peer presence.** Render each peer as a view cone in the 3D scene.
4. **Polish.** Peer-list overlay with names and last-seen, stable per-peer colors,
   throttling.

Key decisions:

- **Potree** for the frontend — the task's recommended path; it handles COPC HTTP-range
  streaming, level-of-detail, and elevation coloring out of the box, so the time went
  into the sync layer rather than into building a renderer. It's pulled in as a git
  submodule so this repo stays to just my own code.
- **Go** for the backend, using the classic hub pattern: a single goroutine owns all
  shared state and everything else talks to it over channels, so there are no mutexes.
  Each connection has a read pump and a write pump with a buffered channel between them,
  so one slow client can't stall the hub.
- **One binary** serves both the static frontend and the `/ws` endpoint — `http.FileServer`
  answers the HTTP range requests COPC needs, and a single origin means no CORS and one
  command to run.
- **A cone** for presence — it communicates both position and view direction in one
  primitive. The apex sits at the peer's eye and it opens along their look direction.
- **Coordinate frames:** peer camera positions are only comparable because every client
  loads the same COPC the same way, so raw world coordinates need no per-client offset.

### What I asked AI to do

I used Claude (in Claude Code) throughout, and drove the decisions myself:

- Up front, to pressure-test the task and surface the traps before coding (the 2 GB
  file, elevation-on-load, disconnect cleanup, and the coordinate-frame issue).
- To confirm the current Potree COPC API against its own `examples/copc.html` rather than
  guessing, and to verify the frontend Potree calls against the submodule source.
- To scaffold the project, write the Go hub boilerplate and the viewer page, and produce
  the architecture diagrams.