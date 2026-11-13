# Geofencing & Real-Time Alert System

Go REST API for geofence management, vehicle tracking, point-in-polygon detection, configurable entry/exit alerts, violation history and WebSocket notifications.

## Features
- Create/list polygon geofences and filter by category
- Register/list vehicles
- Record GPS positions and detect geofence transitions
- Configure per-geofence and optional per-vehicle alert rules
- Persist location history and geofence violations in PostgreSQL
- Broadcast matching entry/exit events over `ws://localhost:8080/ws/alerts`
- Include `time_ns` in API responses
- Docker Compose local environment

## Quick start
```bash
docker compose up --build
```
Open `http://localhost:8080/health` to verify the API.

See [SETUP.md](SETUP.md) for the complete local setup and curl walkthrough.

## Repository layout
- `backend/cmd/api`: application entry point
- `backend/internal/config`: environment configuration
- `backend/internal/database`: PostgreSQL connection and migration runner
- `backend/internal/models`: API/domain models
- `backend/internal/geofence`: polygon validation and ray-casting algorithm
- `backend/internal/websocket`: concurrent WebSocket client hub
- `backend/internal/handlers`: REST handlers and business workflows
- `backend/db/migrations`: SQL schema

## API routes
| Method | Path | Purpose |
|---|---|---|
| POST | `/geofences` | Create polygon geofence |
| GET | `/geofences` | List/filter geofences |
| POST | `/vehicles` | Register vehicle |
| GET | `/vehicles` | List vehicles |
| POST | `/vehicles/location` | Record location and detect transitions |
| GET | `/vehicles/location/{vehicle_id}` | Latest location and current zones |
| POST | `/alerts/configure` | Configure alert rule |
| GET | `/alerts` | List/filter alert rules |
| GET | `/violations/history` | Query event history |
| GET | `/ws/alerts` | WebSocket alert stream |

## Important
This starter is the Go backend portion. The React dashboard, interactive Leaflet map, frontend deployment, Docker Hub publishing and recorded walkthrough must be completed separately before final assessment submission. Do not claim those deliverables are complete until they are implemented and verified.
