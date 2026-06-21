// Realtime peer presence for the Potree viewer.
//
// Connects to the Go WebSocket hub, sends this tab's camera ~once per second, and
// draws every other peer as a colored view cone (apex at their eye position, opening
// along their view direction). One cone per peer, keyed by id, updated in place.

import * as THREE from "../potree/libs/three.js/build/three.module.js";

const SEND_INTERVAL_MS = 1000; // "roughly once per second"

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

export function startPresence({ viewer, coneSize }) {
	const scene = viewer.scene.scene; // THREE scene Potree renders alongside the cloud
	const peers = new Map(); // id -> { mesh }

	function upsertPeer(id, color, camera) {
		if (!camera) return;
		let peer = peers.get(id);
		if (!peer) {
			const mesh = makeCone(color, coneSize);
			scene.add(mesh);
			peer = { mesh };
			peers.set(id, peer);
		}
		placeCone(peer.mesh, camera);
	}

	function removePeer(id) {
		const peer = peers.get(id);
		if (!peer) return;
		scene.remove(peer.mesh);
		peer.mesh.geometry.dispose();
		peer.mesh.material.dispose();
		peers.delete(id);
	}

	function connect() {
		const ws = new WebSocket(`ws://${location.host}/ws`);

		ws.addEventListener("message", (ev) => {
			const msg = JSON.parse(ev.data);
			switch (msg.type) {
				case "init":
					for (const p of msg.peers) upsertPeer(p.id, p.color, p.camera);
					break;
				case "peer":
					upsertPeer(msg.id, msg.color, msg.camera);
					break;
				case "leave":
					removePeer(msg.id);
					break;
			}
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

		ws.addEventListener("close", () => {
			clearInterval(timer);
			// Drop all peer markers and try to reconnect shortly.
			for (const id of [...peers.keys()]) removePeer(id);
			setTimeout(connect, 1000);
		});
	}

	connect();
}
