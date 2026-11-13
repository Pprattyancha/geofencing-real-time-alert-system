# Setup Guide — Geofencing & Real-Time Alert System

## Prerequisites
- Go 1.23+
- Docker Desktop + Docker Compose
- curl or Postman
- Node.js 20+ (for the separately developed React frontend)

## Run with Docker (recommended)
From the repository root:

```bash
docker compose up --build
```

The API is available at `http://localhost:8080`; PostgreSQL is on port `5432`. The backend runs `db/migrations/001_init.sql` on startup. Stop with `Ctrl+C`; remove database data with `docker compose down -v`.

## Run Go API locally
Start PostgreSQL first, create database `geofencing`, then:

```bash
cd backend
cp .env.example .env
# Export DATABASE_URL, PORT and ALLOWED_ORIGIN in your shell (Go does not load .env automatically).
go mod tidy
go test ./...
go run ./cmd/api
```

## API walkthrough

### 1. Create a geofence
```bash
curl -X POST http://localhost:8080/geofences -H 'Content-Type: application/json' -d '{
  "name":"Downtown Restricted Zone",
  "description":"Demo polygon",
  "coordinates":[[37.7749,-122.4194],[37.7849,-122.4194],[37.7849,-122.4094],[37.7749,-122.4094],[37.7749,-122.4194]],
  "category":"restricted_zone"
}'
```
Copy the returned `id` (e.g. `geo_...`).

### 2. Register a vehicle
```bash
curl -X POST http://localhost:8080/vehicles -H 'Content-Type: application/json' -d '{
  "vehicle_number":"KA-01-AB-1234","driver_name":"John Doe","vehicle_type":"truck","phone":"+919876543210"
}'
```
Copy the returned vehicle `id`.

### 3. Configure an entry/exit alert
Replace IDs with the actual IDs returned above:
```bash
curl -X POST http://localhost:8080/alerts/configure -H 'Content-Type: application/json' -d '{
  "geofence_id":"geo_REPLACE","vehicle_id":"veh_REPLACE","event_type":"both"
}'
```

### 4. Connect to WebSocket
Use a WebSocket client at `ws://localhost:8080/ws/alerts` before sending the location update.

### 5. Send a location inside the polygon
```bash
curl -X POST http://localhost:8080/vehicles/location -H 'Content-Type: application/json' -d '{
  "vehicle_id":"veh_REPLACE","latitude":37.7799,"longitude":-122.4144,"timestamp":"2026-11-13T10:00:00Z"
}'
```
You should receive an `entry` message over WebSocket if the alert rule matches. Send a point outside the polygon to trigger `exit`.

### 6. Inspect data
```bash
curl http://localhost:8080/geofences
curl 'http://localhost:8080/geofences?category=restricted_zone'
curl http://localhost:8080/vehicles
curl http://localhost:8080/vehicles/location/veh_REPLACE
curl http://localhost:8080/alerts
curl 'http://localhost:8080/violations/history?vehicle_id=veh_REPLACE&limit=100'
```

## API response timing
Every JSON response includes `time_ns` as a string containing elapsed handler time in nanoseconds. Error responses include it too.

## Frontend integration
The backend exposes REST endpoints on port 8080 and WebSocket on `/ws/alerts`. In Vite, configure `VITE_API_URL=http://localhost:8080` and `VITE_WS_URL=ws://localhost:8080/ws/alerts`. The React frontend is a separate implementation step; this repository currently contains the Go backend foundation only.

## Notes before production
- Configure a strict WebSocket origin allowlist; the starter currently accepts WebSocket origins for local development.
- Replace wildcard CORS with the deployed frontend origin.
- Add authentication/authorization, rate limiting, structured logging, pagination cursors, retention policies and metrics.
- For high-volume fleets, use PostGIS indexes and a message broker (Redis Streams, NATS or Kafka) rather than loading every polygon for each GPS update.
- Use a transactional outbox or durable queue if guaranteed alert delivery is required. The current implementation persists the violation before broadcasting but WebSocket delivery itself is best-effort.
