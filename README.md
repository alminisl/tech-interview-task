# Collaborative Point Cloud Viewer

Multiple browser tabs load the same COPC point cloud and share camera position and
view direction in real time. Each tab shows where the other tabs are looking in 3D via
a view cone.

> Status: work in progress.

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
