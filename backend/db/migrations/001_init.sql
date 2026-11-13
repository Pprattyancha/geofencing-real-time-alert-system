CREATE TABLE IF NOT EXISTS geofences (
 id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '',
 coordinates JSONB NOT NULL, category TEXT NOT NULL CHECK (category IN ('delivery_zone','restricted_zone','toll_zone','customer_area')),
 status TEXT NOT NULL DEFAULT 'active', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS vehicles (
 id TEXT PRIMARY KEY, vehicle_number TEXT NOT NULL UNIQUE, driver_name TEXT NOT NULL,
 vehicle_type TEXT NOT NULL, phone TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'active',
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS vehicle_locations (
 id BIGSERIAL PRIMARY KEY, vehicle_id TEXT NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
 latitude DOUBLE PRECISION NOT NULL, longitude DOUBLE PRECISION NOT NULL,
 recorded_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_vehicle_locations_vehicle_time ON vehicle_locations(vehicle_id, recorded_at DESC);
CREATE TABLE IF NOT EXISTS vehicle_geofence_states (
 vehicle_id TEXT NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
 geofence_id TEXT NOT NULL REFERENCES geofences(id) ON DELETE CASCADE,
 is_inside BOOLEAN NOT NULL DEFAULT FALSE, updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY(vehicle_id, geofence_id)
);
CREATE TABLE IF NOT EXISTS alert_rules (
 id TEXT PRIMARY KEY, geofence_id TEXT NOT NULL REFERENCES geofences(id) ON DELETE CASCADE,
 vehicle_id TEXT REFERENCES vehicles(id) ON DELETE CASCADE,
 event_type TEXT NOT NULL CHECK (event_type IN ('entry','exit','both')),
 status TEXT NOT NULL DEFAULT 'active', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS violations (
 id TEXT PRIMARY KEY, vehicle_id TEXT NOT NULL REFERENCES vehicles(id),
 geofence_id TEXT NOT NULL REFERENCES geofences(id), event_type TEXT NOT NULL,
 latitude DOUBLE PRECISION NOT NULL, longitude DOUBLE PRECISION NOT NULL,
 recorded_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_violations_vehicle ON violations(vehicle_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_violations_geofence ON violations(geofence_id, recorded_at DESC);
