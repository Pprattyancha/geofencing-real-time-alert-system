package geofence

import (
	"errors"
	"github.com/yourusername/geofencing-real-time-alert-system/backend/internal/models"
	"math"
)

var categories = map[string]bool{"delivery_zone": true, "restricted_zone": true, "toll_zone": true, "customer_area": true}

func ValidCategory(c string) bool { return categories[c] }
func ValidatePolygon(points []models.Coordinate) error {
	if len(points) < 4 {
		return errors.New("coordinates must contain at least 4 points including closing point")
	}
	for _, p := range points {
		if math.IsNaN(p[0]) || math.IsNaN(p[1]) || p[0] < -90 || p[0] > 90 || p[1] < -180 || p[1] > 180 {
			return errors.New("coordinate out of range")
		}
	}
	if points[0] != points[len(points)-1] {
		return errors.New("polygon must be closed: first and last coordinates must match")
	}
	unique := map[models.Coordinate]bool{}
	for _, p := range points[:len(points)-1] {
		unique[p] = true
	}
	if len(unique) < 3 {
		return errors.New("polygon must contain at least 3 unique points")
	}
	return nil
}

// Contains uses ray casting. Coordinates are represented as [latitude, longitude].
func Contains(points []models.Coordinate, lat, lon float64) bool {
	inside := false
	for i, j := 0, len(points)-1; i < len(points); j, i = i, i+1 {
		yi, xi := points[i][0], points[i][1]
		yj, xj := points[j][0], points[j][1]
		// Treat a point on an edge as inside.
		cross := (lon-xi)*(yj-yi) - (lat-yi)*(xj-xi)
		if math.Abs(cross) < 1e-10 && lon >= math.Min(xi, xj)-1e-10 && lon <= math.Max(xi, xj)+1e-10 && lat >= math.Min(yi, yj)-1e-10 && lat <= math.Max(yi, yj)+1e-10 {
			return true
		}
		if (yi > lat) != (yj > lat) && lon < (xj-xi)*(lat-yi)/(yj-yi)+xi {
			inside = !inside
		}
	}
	return inside
}
