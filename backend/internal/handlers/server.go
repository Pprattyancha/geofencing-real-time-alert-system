package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourusername/geofencing-real-time-alert-system/backend/internal/geofence"
	"github.com/yourusername/geofencing-real-time-alert-system/backend/internal/models"
	ws "github.com/yourusername/geofencing-real-time-alert-system/backend/internal/websocket"
)

type Server struct {
	  DB          *sql.DB
    Hub           *websocket.Hub
	AllowedOrigin string
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(cors)

	r.Get("/health", s.health)

	// WebSocket alerts endpoint
	r.Get("/ws/alerts", s.Hub.ServeHTTP)

	r.Post("/geofences", s.createGeofence)
	r.Get("/geofences", s.listGeofences)

	r.Post("/vehicles", s.createVehicle)
	r.Get("/vehicles", s.listVehicles)

	r.Post("/vehicles/location", s.updateLocation)
	r.Get("/vehicles/location/{vehicle_id}", s.getLocation)

	r.Post("/alerts/configure", s.configureAlert)
	r.Get("/alerts", s.listAlerts)

	r.Get("/violations/history", s.history)

	return r
}
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func write(w http.ResponseWriter, start time.Time, status int, data any) {
	if m, ok := data.(map[string]any); ok {
		m["time_ns"] = strconv.FormatInt(time.Since(start).Nanoseconds(), 10)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
func fail(w http.ResponseWriter, start time.Time, status int, err error) {
	write(w, start, status, map[string]any{"error": err.Error()})
}
func decode(r *http.Request, v any) error {
	d := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	d.DisallowUnknownFields()
	return d.Decode(v)
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	st := time.Now()
	write(w, st, 200, map[string]any{"status": "ok"})
}
func (s *Server) createGeofence(w http.ResponseWriter, r *http.Request) {
	st := time.Now()
	var in struct {
		Name        string              `json:"name"`
		Description string              `json:"description"`
		Coordinates []models.Coordinate `json:"coordinates"`
		Category    string              `json:"category"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, st, 400, err)
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || !geofence.ValidCategory(in.Category) {
		fail(w, st, 400, errors.New("name and valid category are required"))
		return
	}
	if err := geofence.ValidatePolygon(in.Coordinates); err != nil {
		fail(w, st, 400, err)
		return
	}
	b, _ := json.Marshal(in.Coordinates)
	id := "geo_" + uuid.NewString()
	var created time.Time
	err := s.DB.QueryRow(r.Context(), `INSERT INTO geofences(id,name,description,coordinates,category) VALUES($1,$2,$3,$4,$5) RETURNING created_at`, id, in.Name, in.Description, b, in.Category).Scan(&created)
	if err != nil {
		fail(w, st, 500, err)
		return
	}
	write(w, st, 201, map[string]any{"id": id, "name": in.Name, "status": "active", "created_at": created})
}
func (s *Server) listGeofences(w http.ResponseWriter, r *http.Request) {
	st := time.Now()
	q := `SELECT id,name,description,coordinates,category,status,created_at FROM geofences`
	args := []any{}
	if c := r.URL.Query().Get("category"); c != "" {
		if !geofence.ValidCategory(c) {
			fail(w, st, 400, errors.New("invalid category"))
			return
		}
		q += ` WHERE category=$1`
		args = append(args, c)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := s.DB.Query(r.Context(), q, args...)
	if err != nil {
		fail(w, st, 500, err)
		return
	}
	defer rows.Close()
	items := []models.Geofence{}
	for rows.Next() {
		var x models.Geofence
		var b []byte
		if err = rows.Scan(&x.ID, &x.Name, &x.Description, &b, &x.Category, &x.Status, &x.CreatedAt); err != nil {
			fail(w, st, 500, err)
			return
		}
		_ = json.Unmarshal(b, &x.Coordinates)
		items = append(items, x)
	}
	write(w, st, 200, map[string]any{"geofences": items})
}
func (s *Server) createVehicle(w http.ResponseWriter, r *http.Request) {
	st := time.Now()
	var in struct {
		VehicleNumber string `json:"vehicle_number"`
		DriverName    string `json:"driver_name"`
		VehicleType   string `json:"vehicle_type"`
		Phone         string `json:"phone"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, st, 400, err)
		return
	}
	in.VehicleNumber = strings.TrimSpace(in.VehicleNumber)
	in.DriverName = strings.TrimSpace(in.DriverName)
	in.VehicleType = strings.TrimSpace(in.VehicleType)
	in.Phone = strings.TrimSpace(in.Phone)
	if in.VehicleNumber == "" || in.DriverName == "" || in.VehicleType == "" || in.Phone == "" {
		fail(w, st, 400, errors.New("all vehicle fields are required"))
		return
	}
	id := "veh_" + uuid.NewString()
	var created time.Time
	err := s.DB.QueryRow(r.Context(), `INSERT INTO vehicles(id,vehicle_number,driver_name,vehicle_type,phone) VALUES($1,$2,$3,$4,$5) RETURNING created_at`, id, in.VehicleNumber, in.DriverName, in.VehicleType, in.Phone).Scan(&created)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			fail(w, st, 409, errors.New("vehicle number already exists"))
		} else {
			fail(w, st, 500, err)
		}
		return
	}
	write(w, st, 201, map[string]any{"id": id, "vehicle_number": in.VehicleNumber, "status": "active", "created_at": created})
}
func (s *Server) listVehicles(w http.ResponseWriter, r *http.Request) {
	st := time.Now()
	rows, err := s.DB.Query(r.Context(), `SELECT id,vehicle_number,driver_name,vehicle_type,phone,status,created_at FROM vehicles ORDER BY created_at DESC`)
	if err != nil {
		fail(w, st, 500, err)
		return
	}
	defer rows.Close()
	items := []models.Vehicle{}
	for rows.Next() {
		var v models.Vehicle
		if err = rows.Scan(&v.ID, &v.VehicleNumber, &v.DriverName, &v.VehicleType, &v.Phone, &v.Status, &v.CreatedAt); err != nil {
			fail(w, st, 500, err)
			return
		}
		items = append(items, v)
	}
	write(w, st, 200, map[string]any{"vehicles": items})
}
func (s *Server) updateLocation(w http.ResponseWriter, r *http.Request) {
	st := time.Now()
	var in struct {
		VehicleID string  `json:"vehicle_id"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Timestamp string  `json:"timestamp"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, st, 400, err)
		return
	}
	if in.VehicleID == "" || in.Latitude < -90 || in.Latitude > 90 || in.Longitude < -180 || in.Longitude > 180 {
		fail(w, st, 400, errors.New("valid vehicle_id, latitude and longitude are required"))
		return
	}
	ts, err := time.Parse(time.RFC3339, in.Timestamp)
	if err != nil {
		fail(w, st, 400, errors.New("timestamp must be ISO-8601 RFC3339"))
		return
	}
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		fail(w, st, 500, err)
		return
	}
	defer tx.Rollback(r.Context())
	var vehicle models.AlertVehicle
	if err = tx.QueryRow(r.Context(), `SELECT id,vehicle_number,driver_name FROM vehicles WHERE id=$1 AND status='active' FOR UPDATE`, in.VehicleID).Scan(&vehicle.VehicleID, &vehicle.VehicleNumber, &vehicle.DriverName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, st, 404, errors.New("vehicle not found"))
		} else {
			fail(w, st, 500, err)
		}
		return
	}
	if _, err = tx.Exec(r.Context(), `INSERT INTO vehicle_locations(vehicle_id,latitude,longitude,recorded_at) VALUES($1,$2,$3,$4)`, in.VehicleID, in.Latitude, in.Longitude, ts); err != nil {
		fail(w, st, 500, err)
		return
	}
	rows, err := tx.Query(r.Context(), `SELECT id,name,coordinates,category FROM geofences WHERE status='active'`)
	if err != nil {
		fail(w, st, 500, err)
		return
	}
	type fence struct {
		id, name, category string
		points             []models.Coordinate
	}
	fences := []fence{}
	for rows.Next() {
		var f fence
		var b []byte
		if err = rows.Scan(&f.id, &f.name, &b, &f.category); err != nil {
			rows.Close()
			fail(w, st, 500, err)
			return
		}
		if err = json.Unmarshal(b, &f.points); err != nil {
			rows.Close()
			fail(w, st, 500, err)
			return
		}
		fences = append(fences, f)
	}
	rows.Close()
	current := []models.CurrentGeofence{}
	messages := []models.AlertMessage{}
	for _, f := range fences {
		inside := geofence.Contains(f.points, in.Latitude, in.Longitude)
		var was bool
		err = tx.QueryRow(r.Context(), `SELECT is_inside FROM vehicle_geofence_states WHERE vehicle_id=$1 AND geofence_id=$2`, in.VehicleID, f.id).Scan(&was)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			fail(w, st, 500, err)
			return
		}
		if _, err = tx.Exec(r.Context(), `INSERT INTO vehicle_geofence_states(vehicle_id,geofence_id,is_inside,updated_at) VALUES($1,$2,$3,NOW()) ON CONFLICT(vehicle_id,geofence_id) DO UPDATE SET is_inside=EXCLUDED.is_inside,updated_at=NOW()`, in.VehicleID, f.id, inside); err != nil {
			fail(w, st, 500, err)
			return
		}
		if inside {
			current = append(current, models.CurrentGeofence{GeofenceID: f.id, GeofenceName: f.name, Status: "inside"})
		}
		event := ""
		if !was && inside {
			event = "entry"
		}
		if was && !inside {
			event = "exit"
		}
		if event != "" {
			vid := "viol_" + uuid.NewString()
			if _, err = tx.Exec(r.Context(), `INSERT INTO violations(id,vehicle_id,geofence_id,event_type,latitude,longitude,recorded_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, vid, in.VehicleID, f.id, event, in.Latitude, in.Longitude, ts); err != nil {
				fail(w, st, 500, err)
				return
			}
			ruleRows, e := tx.Query(r.Context(), `SELECT event_type FROM alert_rules WHERE geofence_id=$1 AND status='active' AND (vehicle_id IS NULL OR vehicle_id=$2)`, f.id, in.VehicleID)
			if e != nil {
				fail(w, st, 500, e)
				return
			}
			matched := false
			for ruleRows.Next() {
				var et string
				if e = ruleRows.Scan(&et); e != nil {
					ruleRows.Close()
					fail(w, st, 500, e)
					return
				}
				if et == "both" || et == event {
					matched = true
				}
			}
			ruleRows.Close()
			if matched {
				messages = append(messages, models.AlertMessage{EventID: "evt_" + uuid.NewString(), EventType: event, Timestamp: ts, Vehicle: vehicle, Geofence: models.AlertGeofence{GeofenceID: f.id, GeofenceName: f.name, Category: f.category}, Location: models.AlertLocation{Latitude: in.Latitude, Longitude: in.Longitude}})
			}
		}
	}
	if err = tx.Commit(r.Context()); err != nil {
		fail(w, st, 500, err)
		return
	}
	// Broadcast after commit so clients never receive an event that failed to persist.
	for _, m := range messages {
		go s.Hub.Broadcast(m)
	}
	write(w, st, 200, map[string]any{"vehicle_id": in.VehicleID, "location_updated": true, "current_geofences": current})
}
func (s *Server) getLocation(w http.ResponseWriter, r *http.Request) {
	st := time.Now()
	id := chi.URLParam(r, "vehicle_id")
	var v models.Vehicle
	err := s.DB.QueryRow(r.Context(), `SELECT id,vehicle_number,driver_name,vehicle_type,phone,status,created_at FROM vehicles WHERE id=$1`, id).Scan(&v.ID, &v.VehicleNumber, &v.DriverName, &v.VehicleType, &v.Phone, &v.Status, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, st, 404, errors.New("vehicle not found"))
		return
	}
	if err != nil {
		fail(w, st, 500, err)
		return
	}
	var loc models.Location
	err = s.DB.QueryRow(r.Context(), `SELECT latitude,longitude,recorded_at FROM vehicle_locations WHERE vehicle_id=$1 ORDER BY recorded_at DESC,id DESC LIMIT 1`, id).Scan(&loc.Latitude, &loc.Longitude, &loc.Timestamp)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, st, 404, errors.New("no location recorded for vehicle"))
		return
	}
	if err != nil {
		fail(w, st, 500, err)
		return
	}
	rows, err := s.DB.Query(r.Context(), `SELECT g.id,g.name,g.category FROM vehicle_geofence_states st JOIN geofences g ON g.id=st.geofence_id WHERE st.vehicle_id=$1 AND st.is_inside=true`, id)
	if err != nil {
		fail(w, st, 500, err)
		return
	}
	defer rows.Close()
	current := []models.CurrentGeofence{}
	for rows.Next() {
		var x models.CurrentGeofence
		if err = rows.Scan(&x.GeofenceID, &x.GeofenceName, &x.Category); err != nil {
			fail(w, st, 500, err)
			return
		}
		current = append(current, x)
	}
	write(w, st, 200, map[string]any{"vehicle_id": id, "vehicle_number": v.VehicleNumber, "current_location": loc, "current_geofences": current})
}
func (s *Server) configureAlert(w http.ResponseWriter, r *http.Request) {
	st := time.Now()
	var in struct {
		GeofenceID string  `json:"geofence_id"`
		VehicleID  *string `json:"vehicle_id"`
		EventType  string  `json:"event_type"`
	}
	if err := decode(r, &in); err != nil {
		fail(w, st, 400, err)
		return
	}
	if in.GeofenceID == "" || (in.EventType != "entry" && in.EventType != "exit" && in.EventType != "both") {
		fail(w, st, 400, errors.New("geofence_id and event_type (entry, exit, both) are required"))
		return
	}
	id := "alert_" + uuid.NewString()
	var created time.Time
	err := s.DB.QueryRow(r.Context(), `INSERT INTO alert_rules(id,geofence_id,vehicle_id,event_type) VALUES($1,$2,$3,$4) RETURNING created_at`, id, in.GeofenceID, in.VehicleID, in.EventType).Scan(&created)
	if err != nil {
		fail(w, st, 400, fmt.Errorf("could not configure alert: %w", err))
		return
	}
	write(w, st, 201, map[string]any{"alert_id": id, "geofence_id": in.GeofenceID, "vehicle_id": in.VehicleID, "event_type": in.EventType, "status": "active", "created_at": created})
}
func (s *Server) listAlerts(w http.ResponseWriter, r *http.Request) {
	st := time.Now()
	q := `SELECT a.id,a.geofence_id,g.name,a.vehicle_id,v.vehicle_number,a.event_type,a.status,a.created_at FROM alert_rules a JOIN geofences g ON g.id=a.geofence_id LEFT JOIN vehicles v ON v.id=a.vehicle_id WHERE 1=1`
	args := []any{}
	if x := r.URL.Query().Get("geofence_id"); x != "" {
		args = append(args, x)
		q += fmt.Sprintf(" AND a.geofence_id=$%d", len(args))
	}
	if x := r.URL.Query().Get("vehicle_id"); x != "" {
		args = append(args, x)
		q += fmt.Sprintf(" AND a.vehicle_id=$%d", len(args))
	}
	q += " ORDER BY a.created_at DESC"
	rows, err := s.DB.Query(r.Context(), q, args...)
	if err != nil {
		fail(w, st, 500, err)
		return
	}
	defer rows.Close()
	out := []models.AlertRule{}
	for rows.Next() {
		var a models.AlertRule
		if err = rows.Scan(&a.ID, &a.GeofenceID, &a.GeofenceName, &a.VehicleID, &a.VehicleNumber, &a.EventType, &a.Status, &a.CreatedAt); err != nil {
			fail(w, st, 500, err)
			return
		}
		out = append(out, a)
	}
	write(w, st, 200, map[string]any{"alerts": out})
}
func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	st := time.Now()
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, e := strconv.Atoi(raw)
		if e != nil || n < 1 {
			fail(w, st, 400, errors.New("limit must be a positive integer"))
			return
		}
		if n > 500 {
			n = 500
		}
		limit = n
	}
	q := `SELECT x.id,x.vehicle_id,v.vehicle_number,x.geofence_id,g.name,x.event_type,x.latitude,x.longitude,x.recorded_at FROM violations x JOIN vehicles v ON v.id=x.vehicle_id JOIN geofences g ON g.id=x.geofence_id WHERE 1=1`
	args := []any{}
	add := func(key, clause string) {
		if val := r.URL.Query().Get(key); val != "" {
			args = append(args, val)
			q += fmt.Sprintf(" AND %s=$%d", clause, len(args))
		}
	}
	add("vehicle_id", "x.vehicle_id")
	add("geofence_id", "x.geofence_id")
	for _, f := range []struct{ k, op string }{{"start_date", ">="}, {"end_date", "<="}} {
		if val := r.URL.Query().Get(f.k); val != "" {
			t, e := time.Parse(time.RFC3339, val)
			if e != nil {
				fail(w, st, 400, fmt.Errorf("%s must be RFC3339", f.k))
				return
			}
			args = append(args, t)
			q += fmt.Sprintf(" AND x.recorded_at %s $%d", f.op, len(args))
		}
	}
	countQ := strings.Replace(q, "SELECT x.id,x.vehicle_id,v.vehicle_number,x.geofence_id,g.name,x.event_type,x.latitude,x.longitude,x.recorded_at", "SELECT COUNT(*)", 1)
	var total int
	if err := s.DB.QueryRow(r.Context(), countQ, args...).Scan(&total); err != nil {
		fail(w, st, 500, err)
		return
	}
	args = append(args, limit)
	q += fmt.Sprintf(" ORDER BY x.recorded_at DESC LIMIT $%d", len(args))
	rows, err := s.DB.Query(r.Context(), q, args...)
	if err != nil {
		fail(w, st, 500, err)
		return
	}
	defer rows.Close()
	out := []models.Violation{}
	for rows.Next() {
		var x models.Violation
		if err = rows.Scan(&x.ID, &x.VehicleID, &x.VehicleNumber, &x.GeofenceID, &x.GeofenceName, &x.EventType, &x.Latitude, &x.Longitude, &x.Timestamp); err != nil {
			fail(w, st, 500, err)
			return
		}
		out = append(out, x)
	}
	write(w, st, 200, map[string]any{"violations": out, "total_count": total})
}
