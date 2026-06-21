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

_(to be documented)_

## How to run

_(to be documented)_

## How to test with two browser tabs

_(to be documented)_

## Approach

See [AI_USAGE.md](./AI_USAGE.md) for the full phase-by-phase log of how this was built
and how AI was used. A condensed Approach section will land here.
