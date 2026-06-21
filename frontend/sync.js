// Realtime peer presence for the Potree viewer.
//
// Connects to the Go WebSocket hub, sends this tab's camera ~once per second, and
// draws every other peer as a colored view cone (apex at their eye position, opening
// along their view direction). Also maintains a top-right overlay listing connected
// peers with their color, name and last-seen time, plus an input to set your own name.
//
// One cone and one list row per peer, keyed by id, updated in place.

import * as THREE from "../potree/libs/three.js/build/three.module.js";

const SEND_INTERVAL_MS = 1000; // "roughly once per second"
const NAME_KEY = "pcv-name"; // localStorage key so a chosen name survives refresh

// Build a cone whose apex sits at the origin and opens along -Y, so we can point it
// by rotating -Y onto the peer's view direction and placing the apex at their eye.
function makeCone(color, size) {
	const geometry = new THREE.ConeGeometry(size * 0.5, size, 16, 1, true);
	geometry.translate(0, -size / 2, 0); // move apex to the origin
	const material = new THREE.MeshBasicMaterial({
		color: new THREE.Color(color),
		transparent: true,
		opacity: 0.35,
		side: THREE.DoubleSide,
		depthWrite: false,
	});
	const mesh = new THREE.Mesh(geometry, material);
	// A wireframe edge makes the cone readable against a dense point cloud.
	const edges = new THREE.Mesh(
		geometry,
		new THREE.MeshBasicMaterial({ color: new THREE.Color(color), wireframe: true })
	);
	mesh.add(edges);
	return mesh;
}

const DOWN = new THREE.Vector3(0, -1, 0);

// Orient + place a cone from a camera state {position, direction}.
function placeCone(mesh, camera) {
	mesh.position.set(camera.position.x, camera.position.y, camera.position.z);
	const dir = new THREE.Vector3(
		camera.direction.x,
		camera.direction.y,
		camera.direction.z
	).normalize();
	mesh.quaternion.setFromUnitVectors(DOWN, dir);
}

// Human-readable "time since" for the last-seen column.
function relTime(ts) {
	if (!ts) return "";
	const s = Math.floor((Date.now() - ts) / 1000);
	if (s < 2) return "just now";
	if (s < 60) return `${s}s ago`;
	return `${Math.floor(s / 60)}m ago`;
}

function dot(color) {
	return `<span style="display:inline-block;width:10px;height:10px;border-radius:50%;background:${color};margin-right:8px;"></span>`;
}

export function startPresence({ viewer, coneSize }) {
	const scene = viewer.scene.scene; // THREE scene Potree renders alongside the cloud
	const peers = new Map(); // id -> { color, name, mesh?, lastSeen? }

	const listEl = document.getElementById("peer-list");
	const titleEl = document.getElementById("presence-title");
	const nameInput = document.getElementById("name-input");

	// This tab's own identity, learned from the server's init message.
	const self = { id: null, color: "#ffffff", name: localStorage.getItem(NAME_KEY) || "" };
	if (self.name) nameInput.value = self.name;

	function ensurePeer(id, color, name) {
		let peer = peers.get(id);
		if (!peer) {
			peer = { color, name };
			peers.set(id, peer);
		}
		if (color) peer.color = color;
		if (name) peer.name = name;
		return peer;
	}

	function updateCone(id, camera) {
		if (!camera) return;
		const peer = peers.get(id);
		if (!peer) return;
		if (!peer.mesh) {
			peer.mesh = makeCone(peer.color, coneSize);
			scene.add(peer.mesh);
		}
		placeCone(peer.mesh, camera);
		peer.lastSeen = Date.now();
	}

	function removePeer(id) {
		const peer = peers.get(id);
		if (!peer) return;
		if (peer.mesh) {
			scene.remove(peer.mesh);
			peer.mesh.geometry.dispose();
			peer.mesh.material.dispose();
		}
		peers.delete(id);
	}

	function renderList() {
		const rows = [];
		// Self first, marked, using the color the server assigned us.
		const selfName = self.name || self.id || "you";
		rows.push(`<li style="margin:3px 0;">${dot(self.color)}${selfName} <em style="opacity:0.6;">(you)</em></li>`);
		for (const [id, peer] of peers) {
			const label = peer.name || id;
			const seen = relTime(peer.lastSeen);
			rows.push(
				`<li style="margin:3px 0;display:flex;justify-content:space-between;gap:10px;">` +
				`<span>${dot(peer.color)}${label}</span>` +
				`<span style="opacity:0.6;">${seen}</span></li>`
			);
		}
		listEl.innerHTML = rows.join("");
		titleEl.textContent = `Connected (${peers.size + 1})`;
	}

	function connect() {
		const ws = new WebSocket(`ws://${location.host}/ws`);

		ws.addEventListener("open", () => {
			// Re-assert our chosen name on (re)connect, since the server assigns a new id.
			if (self.name) ws.send(JSON.stringify({ type: "name", name: self.name }));
		});

		ws.addEventListener("message", (ev) => {
			const msg = JSON.parse(ev.data);
			switch (msg.type) {
				case "init":
					self.id = msg.id;
					self.color = msg.color;
					for (const p of msg.peers) {
						ensurePeer(p.id, p.color, p.name);
						if (p.camera) updateCone(p.id, p.camera);
						else peers.get(p.id).lastSeen = Date.now();
					}
					break;
				case "peer":
					ensurePeer(msg.id, msg.color, msg.name);
					updateCone(msg.id, msg.camera);
					break;
				case "name": {
					const peer = peers.get(msg.id);
					if (peer) peer.name = msg.name;
					break;
				}
				case "leave":
					removePeer(msg.id);
					break;
			}
			renderList();
		});

		// Send our camera at a steady ~1 Hz, but skip frames where nothing moved so we
		// don't flood the hub while the user is idle.
		let last = "";
		const timer = setInterval(() => {
			if (ws.readyState !== WebSocket.OPEN) return;
			const cam = viewer.scene.getActiveCamera();
			const pos = cam.getWorldPosition(new THREE.Vector3());
			const dir = cam.getWorldDirection(new THREE.Vector3());
			const camera = {
				position: { x: pos.x, y: pos.y, z: pos.z },
				direction: { x: dir.x, y: dir.y, z: dir.z },
			};
			const encoded = JSON.stringify(camera);
			if (encoded === last) return;
			last = encoded;
			ws.send(JSON.stringify({ type: "camera", camera }));
		}, SEND_INTERVAL_MS);

		// Let the user set a display name; persist it and tell the server.
		nameInput.onchange = () => {
			self.name = nameInput.value.trim().slice(0, 32);
			localStorage.setItem(NAME_KEY, self.name);
			if (ws.readyState === WebSocket.OPEN) {
				ws.send(JSON.stringify({ type: "name", name: self.name }));
			}
			renderList();
		};

		ws.addEventListener("close", () => {
			clearInterval(timer);
			// Drop all peer markers and try to reconnect shortly.
			for (const id of [...peers.keys()]) removePeer(id);
			renderList();
			setTimeout(connect, 1000);
		});
	}

	// Keep the last-seen times fresh even when no messages arrive.
	setInterval(renderList, 1000);
	renderList();
	connect();
}
