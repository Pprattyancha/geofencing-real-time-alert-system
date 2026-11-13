# GeoPulse — Geofencing & Real-time Alert System (Frontend)

React + TypeScript + Vite frontend for the MapUp geofencing assessment.

## Features
- Overview dashboard with API and WebSocket connection indicators
- Geofence list, category filter, and Leaflet map polygons
- Create geofences using coordinate pairs
- Register vehicles
- Send vehicle location updates to the Go API
- Configure entry/exit alert rules
- Live WebSocket alert feed with reconnect behavior
- Violation history with vehicle/geofence filters
- Responsive dark operations dashboard

## Requirements
- Node.js 20+
- npm
- Go backend and PostgreSQL running (the backend should be reachable at `http://localhost:8080`)

## Run locally
1. Open a terminal in this `frontend` folder.
2. Install packages:
   ```bash
   npm install
   ```
3. Copy `.env.example` to `.env` (optional; defaults to localhost):
   ```env
   VITE_API_URL=http://localhost:8080
   ```
4. Start the frontend:
   ```bash
   npm run dev
   ```
5. Open the URL printed by Vite, usually `http://localhost:5173`.

## Build
```bash
npm run build
npm run preview
```

## Backend integration
The frontend uses the exact assessment routes:
- `GET/POST /geofences`
- `GET/POST /vehicles`
- `POST /vehicles/location`
- `POST /alerts/configure`
- `GET /alerts`
- `GET /violations/history`
- `GET /ws/alerts` (WebSocket)

All requests use `VITE_API_URL`. The WebSocket URL is derived from it.

## Important implementation notes
- Geofence coordinates must be entered as `latitude,longitude`, one pair per line. The first and last pairs must be identical and the polygon must contain at least four pairs.
- The map is powered by OpenStreetMap tiles through Leaflet. The initial map viewport is India; zoom/pan to your area.
- The current backend API does not expose a vehicle-location listing endpoint. The map therefore visualizes saved geofence polygons; vehicle location updates are submitted through the form and the API response is refreshed into the dashboard. Add a backend endpoint if you want persistent vehicle markers to be fetched and rendered accurately.
- WebSocket alerts are displayed as soon as the backend broadcasts them. If the backend is not running, the dashboard shows API unavailable and the WebSocket reconnects automatically.

## Suggested local workflow
1. Start PostgreSQL and Go backend from the repository root using `docker compose up --build`.
2. In a second terminal, run this frontend with `npm install` and `npm run dev`.
3. Create a geofence.
4. Register a vehicle.
5. Configure an alert rule for that geofence and vehicle (or all vehicles).
6. Send a location update inside/outside the polygon to trigger events.
7. Inspect the live alert feed and violation history.

## Deployment
Deploy this folder to Vercel, Netlify, or Cloudflare Pages. Set `VITE_API_URL` to the public backend URL. Ensure the backend permits the deployed frontend origin and the WebSocket endpoint is reachable over `wss://`.
